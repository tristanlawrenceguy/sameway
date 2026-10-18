package agenda_test

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

// At the end of the day, Today says what got done, by name; before
// anything is done it says nothing of it.
func TestTodaySaysWhatGotDone(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	if page := get(t, h, "/today").Body.String(); strings.Contains(page, "Done today") {
		t.Error("nothing done yet, nothing said")
	}
	rent, _ := a.Store.Create("task", map[string]any{"title": "Pay rent", "due": tonight.Format("2006-01-02")})
	postForm(t, h, "/today/done", url.Values{"id": {rent.ID}})
	if page := get(t, h, "/today").Body.String(); !strings.Contains(page, "Done today: Pay rent.") {
		t.Errorf("the task ticked today is said: %s", truncate(page))
	}
	if api := get(t, h, "/api/today").Body.String(); !strings.Contains(api, `"done_today"`) || !strings.Contains(api, "Pay rent") {
		t.Errorf("an agent reads it too: %s", truncate(api))
	}
}

// Done today is more than tasks: a daily habit met today, which leaves
// Today's habits once met, and a call that happened today are said too.
func TestDoneTodayHasHabitsMetAndCalls(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	pills, _ := a.Store.Create("habit", map[string]any{"name": "Mum's tablets"})
	a.Store.Create("entry", map[string]any{"habit": pills.ID, "amount": 1, "at": tonight.Format("2006-01-02")})
	a.Store.Create("interaction", map[string]any{"summary": "Called the GP", "at": tonight.Add(-2 * time.Hour).Format(time.RFC3339)})
	page := get(t, h, "/today").Body.String()
	if !strings.Contains(page, "Done today: Mum&#39;s tablets and Called the GP.") {
		t.Errorf("habits met and calls are said: %s", truncate(page))
	}
}
