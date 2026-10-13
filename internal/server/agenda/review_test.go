package agenda_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// The weekly review says the week in one sentence, done first; shows what
// was done, what is overdue (each moved to next week, kept with no date or
// ticked in one press), the week ahead with what is on the calendar; and
// on the day set, in the evening, says once that it is ready, with what
// was done and never what is late.
func TestTheWeeklyReview(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	now := time.Now()
	day := func(d int) string { return now.AddDate(0, 0, d).Format("2006-01-02") }
	ids := map[string]string{}
	for _, f := range []map[string]any{
		{"title": "Pay the plumber", "due": day(-1), "done": true},
		{"title": "Renew passport", "due": day(-3)},
		{"title": "Call the bank", "due": day(-2)},
		{"title": "Learn the cello", "due": day(-4)},
		{"title": "Send the invoice", "due": day(2)},
		{"title": "Plan the holiday", "due": day(30)},
	} {
		rec, err := a.Store.Create("task", f)
		if err != nil {
			t.Fatal(err)
		}
		ids[f["title"].(string)] = rec.ID
	}
	for _, f := range []map[string]any{{"title": "Dentist", "starts": day(3)}, {"title": "Wedding", "starts": day(40)}} {
		if _, err := a.Store.Create("event", f); err != nil {
			t.Fatal(err)
		}
	}
	h := server.New(a)
	page := get(t, h, "/review").Body.String()
	for _, want := range []string{"1 task done, 3 overdue and 1 due in the week ahead.", "Done this week", "Pay the plumber", "<h2>Overdue</h2>", "Renew passport", "Clear the date", "The week ahead", "Send the invoice", "On the calendar", "Dentist", "Look back and plan with the assistant"} {
		if !strings.Contains(page, want) {
			t.Errorf("the review has %s", want)
		}
	}
	for _, not := range []string{"Plan the holiday", "Wedding", "Slipped", "Habits"} {
		if strings.Contains(page, not) {
			t.Errorf("the review leaves out %s: next month is not the week ahead, and no habits is not a section", not)
		}
	}
	postForm(t, h, "/review/next", url.Values{"id": {ids["Renew passport"]}})
	postForm(t, h, "/review/done", url.Values{"id": {ids["Call the bank"]}})
	postForm(t, h, "/review/undate", url.Values{"id": {ids["Learn the cello"]}})
	moved, _ := a.Store.Get("task", ids["Renew passport"])
	if !strings.HasPrefix(moved.Fields["due"].(string), day(7)) {
		t.Errorf("moved to next week: %v", moved.Fields["due"])
	}
	if bank, _ := a.Store.Get("task", ids["Call the bank"]); bank.Fields["done"] != true {
		t.Error("ticked")
	}
	if cello, _ := a.Store.Get("task", ids["Learn the cello"]); cello.Fields["due"] != nil && cello.Fields["due"] != "" || cello.Fields["done"] == true {
		t.Errorf("kept, with no date: %v", cello.Fields)
	}
	if _, err := a.Store.Create("habit", map[string]any{"name": "Stretch"}); err != nil {
		t.Fatal(err)
	}
	page = get(t, h, "/review").Body.String()
	if strings.Contains(page, "<h2>Overdue</h2>") {
		t.Error("with nothing overdue, the review has no Overdue section")
	}
	if strings.Count(page, ">Habits</h2>") != 1 || !strings.Contains(page, `aria-label="Habits"`) || !strings.Contains(page, "Stretch") {
		t.Error("habits kept are one section, its heading and its region one name")
	}
	var sent []string
	h.OnRing(func(title, text, link string) { sent = append(sent, title+" | "+text+" | "+link) })
	postForm(t, h, "/review/on", url.Values{"on": {strings.ToLower(now.Weekday().String())}})
	evening := time.Date(now.Year(), now.Month(), now.Day(), 18, 30, 0, 0, time.Local)
	if !h.ReviewIfDue(evening) || h.ReviewIfDue(evening) {
		t.Fatal("ready once on the day, in the evening")
	}
	time.Sleep(50 * time.Millisecond)
	if len(sent) != 1 || !strings.HasSuffix(sent[0], "/review") || !strings.Contains(sent[0], "2 tasks done this week") || strings.Contains(sent[0], "overdue") {
		t.Errorf("it says what was done, not what is late, leading to the review: %v", sent)
	}
}

// The calendar in the week ahead crosses a month's end: on 28 October a
// meeting on 2 November is in the next seven days, one on 9 November is
// not.
func TestTheReviewCalendarCrossesTheMonth(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 28, 9, 0, 0, 0, time.Local)
	a, h := newAppWith(t, app.Options{Clock: func() time.Time { return now }})
	for _, f := range []map[string]any{{"title": "Board meeting", "starts": "2026-11-02"}, {"title": "Wedding", "starts": "2026-11-09"}} {
		if _, err := a.Store.Create("event", f); err != nil {
			t.Fatal(err)
		}
	}
	page := get(t, h, "/review").Body.String()
	if !strings.Contains(page, "Board meeting") || strings.Contains(page, "Wedding") {
		t.Errorf("the next seven days, into November\n%s", truncate(page))
	}
}
