package agenda_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server/servertest"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A long Today starts with three, what matters first, the rest folded
// away; a short one is shown whole. Everything in the person's head, put
// in the box, goes to the assistant to sort.
func TestALongTodayStartsWithThree(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	now := tonight
	day := func(n int) string { return now.AddDate(0, 0, n).Format("2006-01-02") }
	for i, title := range []string{"Renew passport", "Pay rent", "Call the bank", "Book dentist"} {
		a.Store.Create("task", map[string]any{"title": title, "due": day(-1 - i)})
	}
	if page := get(t, h, "/today").Body.String(); strings.Contains(page, "Start with these") || !strings.Contains(page, "Too much in your head?</h2>") {
		t.Fatalf("four is read whole, with the box: %s", truncate(page))
	}
	a.Store.Create("task", map[string]any{"title": "Reply to the school", "due": day(0)})
	a.Store.Create("task", map[string]any{"title": "Send the tax form", "due": day(0), "tags": []any{"important"}})
	page := get(t, h, "/today").Body.String()
	start := strings.Index(page, "Start with these")
	rest := strings.Index(page, "The rest")
	if start < 0 || rest < start {
		t.Fatalf("six start with three, the rest after: %s", truncate(page))
	}
	first := page[start:rest]
	for _, want := range []string{"Send the tax form", "Book dentist", "Call the bank"} {
		if !strings.Contains(first, want) {
			t.Errorf("what matters, then the latest late, start: %q missing from %s", want, first)
		}
	}
	if strings.Index(first, "Send the tax form") > strings.Index(first, "Book dentist") {
		t.Error("what matters comes first")
	}
	if !strings.Contains(page[rest:], "Renew passport") {
		t.Error("the rest are there, folded")
	}

	postForm(t, h, "/today/sort", url.Values{"words": {"dentist friday, call mum, taxes due 31st"}})
	msgs, _ := a.Store.List("message", store.ListOptions{})
	asked := false
	for _, m := range msgs {
		asked = asked || m.Fields["role"] == "user" && strings.Contains(m.Fields["content"].(string), "call mum") && strings.Contains(m.Fields["content"].(string), "three to start with")
	}
	if !asked {
		t.Error("the box goes to the assistant, asked to sort it")
	}
}

// What the person marked as mattering joins the three to start with
// before it is due: told by the assistant to start with the dentist and
// the council tax, they found Today starting with errands from a fortnight
// ago. It never joins the rest, and the count stays what is late or due.
// Sorting what is in their head brings them back to Today.
func TestWhatMattersStartsBeforeItIsDue(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	day := func(n int) string { return tonight.AddDate(0, 0, n).Format("2006-01-02") }
	for i, title := range []string{"Buy flowers", "Buy fresh produce", "Return library books", "Pick up dry cleaning", "Water the plants", "Clean the fridge"} {
		a.Store.Create("task", map[string]any{"title": title, "due": day(-14 + i)})
	}
	a.Store.Create("task", map[string]any{"title": "Move the dentist appointment", "due": day(2), "tags": []any{"important"}})
	a.Store.Create("task", map[string]any{"title": "Pay the council tax", "tags": []any{"important"}})
	a.Store.Create("task", map[string]any{"title": "Plan the holiday", "due": day(30), "tags": []any{"important"}})
	page := get(t, h, "/today").Body.String()
	start, rest := strings.Index(page, "Start with these"), strings.Index(page, "The rest")
	if start < 0 || rest < start {
		t.Fatalf("six late start with three: %s", truncate(page))
	}
	first := page[start:rest]
	for _, want := range []string{"Move the dentist appointment", "Pay the council tax"} {
		if !strings.Contains(first, want) {
			t.Errorf("what matters starts, due or not: %q missing", want)
		}
	}
	if strings.Index(first, "Move the dentist appointment") > strings.Index(first, "Pay the council tax") {
		t.Error("one with a day comes before one with none")
	}
	if strings.Contains(page, "Plan the holiday") {
		t.Error("what matters a month from now is not today's")
	}
	if strings.Contains(page[rest:], "Pay the council tax") || !strings.Contains(page, "6 tasks are late or due.") {
		t.Errorf("never in the rest, and the count is what is late or due")
	}
	if res := postForm(t, h, "/today/sort", url.Values{"words": {"call mum"}}); !strings.HasPrefix(res.Header().Get("Location"), "/today") {
		t.Errorf("sorted, the person is back on Today: %q", res.Header().Get("Location"))
	}
}

// On a long Today, tasks late more than two weeks are offered, above the
// three, to keep with no date in one press: no longer late, still tasks, all back
// with one Undo; what matters is never among them.
func TestLongLateTasksCanBeKeptWithNoDate(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	day := func(n int) string { return tonight.AddDate(0, 0, n).Format("2006-01-02") }
	var old []string
	for i, title := range []string{"Buy flowers", "Buy fresh produce", "Return library books"} {
		r, _ := a.Store.Create("task", map[string]any{"title": title, "due": day(-20 - i)})
		old = append(old, r.ID)
	}
	keep, _ := a.Store.Create("task", map[string]any{"title": "File the tax return", "due": day(-30), "tags": []any{"important"}})
	for i, title := range []string{"Water the plants", "Clean the fridge", "Call the plumber", "Pay rent"} {
		a.Store.Create("task", map[string]any{"title": title, "due": day(-1 - i)})
	}
	page := get(t, h, "/today").Body.String()
	if !strings.Contains(page, "3 of them have been late more than two weeks.") || !strings.Contains(page, "Keep them, with no date") {
		t.Fatalf("the long late are offered: %s", truncate(page))
	}
	postForm(t, h, "/today/let-go", url.Values{})
	for _, id := range old {
		if r, _ := a.Store.Get("task", id); r.Fields["due"] != nil && r.Fields["due"] != "" {
			t.Errorf("kept with no date: %v", r.Fields["due"])
		}
	}
	if r, _ := a.Store.Get("task", keep.ID); r.Fields["due"] == nil || r.Fields["due"] == "" {
		t.Error("what matters keeps its day")
	}
	postForm(t, h, "/activity/"+servertest.NewestEntry(t, a.Store).ID+"/undo", url.Values{})
	if r, _ := a.Store.Get("task", old[0]); r.Fields["due"] == nil || r.Fields["due"] == "" {
		t.Error("one Undo puts their days back")
	}
}

// The tasks the assistant said to start with lead Today's three: tagged
// start, one with no day comes before what matters with a day, so Today
// and the assistant give the person one answer.
func TestTodayStartsWithWhatTheAssistantSaid(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	day := func(n int) string { return tonight.AddDate(0, 0, n).Format("2006-01-02") }
	for i, title := range []string{"Buy flowers", "Return library books", "Water the plants", "Clean the fridge"} {
		a.Store.Create("task", map[string]any{"title": title, "due": day(-7 + i), "tags": []any{"important"}})
	}
	a.Store.Create("task", map[string]any{"title": "Pick up dry cleaning", "due": day(-2)})
	a.Store.Create("task", map[string]any{"title": "Wash the car", "due": day(-1)})
	a.Store.Create("task", map[string]any{"title": "Call the dentist", "tags": []any{"start"}})
	page := get(t, h, "/today").Body.String()
	start, rest := strings.Index(page, "Start with these"), strings.Index(page, "The rest")
	if start < 0 || rest < start {
		t.Fatalf("a long Today starts with three: %s", truncate(page))
	}
	first := page[start:rest]
	if dentist := strings.Index(first, "Call the dentist"); dentist < 0 || dentist > strings.Index(first, "Buy flowers") {
		t.Errorf("the task tagged start leads the three:\n%s", first)
	}
}
