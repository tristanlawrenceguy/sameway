package agenda_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// What the assistant said to start with leads Today on a short day too:
// told to start with the dentist, a person with two tasks due found Today
// without it. Only it comes first, the rest stay unfolded under their own
// headings, and the morning brief names the same task.
func TestAShortTodayStartsWithWhatTheAssistantSaid(t *testing.T) {
	t.Parallel()
	a, _ := newAppAt(t, tonight)
	h := server.New(a)
	day := func(n int) string { return tonight.AddDate(0, 0, n).Format("2006-01-02") }
	a.Store.Create("task", map[string]any{"title": "Water the plants", "due": day(-2)})
	a.Store.Create("task", map[string]any{"title": "Pay rent", "due": day(0)})
	a.Store.Create("task", map[string]any{"title": "Book the boiler service", "due": day(20), "tags": []any{"start"}})
	page := get(t, h, "/today").Body.String()
	start, late, due := strings.Index(page, "<h2>Start with this</h2>"), strings.Index(page, "<h2>Overdue</h2>"), strings.Index(page, "<h2>Tasks</h2>")
	if start < 0 || late < start || due < late || !strings.Contains(page, "The assistant said to start with this.") {
		t.Fatalf("the task tagged start comes first, the rest as usual after: %s", truncate(page))
	}
	if !strings.Contains(page[start:late], "Book the boiler service") || strings.Contains(page, "The rest") {
		t.Error("a start task leads whatever its day, and nothing is folded on a short day")
	}
	sent := make(chan string, 1)
	h.OnRing(func(title, text, link string) { sent <- text })
	postForm(t, h, "/brief", url.Values{"at": {"00:00"}})
	if !h.BriefIfDue(tonight) {
		t.Fatal("a day with tasks due sends the brief")
	}
	if got := <-sent; got != "Start with Book the boiler service" {
		t.Errorf("the brief says what Today says: %q", got)
	}
	if task := get(t, h, "/api/describe/types/task").Body.String(); !strings.Contains(task, "Tagged start, it leads Today") {
		t.Error("an agent reading describe learns the start tag the assistant is told about")
	}
}

// When the assistant's picks and what is marked important both lead, the
// line above the three says which came first, so they never seem to
// disagree; a short day with nothing said is read whole, as before.
func TestTheLineSaysWhoseOrderTheThreeFollow(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	day := func(n int) string { return tonight.AddDate(0, 0, n).Format("2006-01-02") }
	for i, title := range []string{"Buy flowers", "Return library books", "Water the plants", "Clean the fridge"} {
		a.Store.Create("task", map[string]any{"title": title, "due": day(-7 + i)})
	}
	if page := get(t, h, "/today").Body.String(); strings.Contains(page, "Start with") {
		t.Errorf("a short day with nothing said to start with is read whole: %s", truncate(page))
	}
	a.Store.Create("task", map[string]any{"title": "Pick up dry cleaning", "due": day(-2), "tags": []any{"important"}})
	a.Store.Create("task", map[string]any{"title": "Wash the car", "due": day(-1)})
	a.Store.Create("task", map[string]any{"title": "Call the dentist", "tags": []any{"start"}})
	page := get(t, h, "/today").Body.String()
	if !strings.Contains(page, "6 tasks are late or due. Start with these three, what the assistant said first, then what is marked important; the rest wait below.") {
		t.Errorf("the line says the assistant's pick leads, then what is marked: %s", truncate(page))
	}
}
