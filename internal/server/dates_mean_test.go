package server_test

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

// Only what can be done is late: a log of habit entries, or a meeting that
// has passed, is Earlier, the latest first, while a task past its day is
// Overdue; two records alike are told apart by their day as its field
// says it (due for a task, the time alone for an entry); and a group's
// count is not followed by a comma with nothing after it.
func TestOnlyWhatCanBeDoneIsOverdue(t *testing.T) {
	a, h := newAppAt(t, tonight)
	now := tonight
	hb, _ := a.Store.Create("habit", map[string]any{"name": "Read", "unit": "minutes", "cadence": "day", "target": 30})
	for i := 1; i <= 3; i++ {
		a.Store.Create("entry", map[string]any{"habit": hb.ID, "amount": 20, "at": now.AddDate(0, 0, -i).Format(time.RFC3339)})
	}
	a.Store.Create("task", map[string]any{"title": "Pay rent", "due": now.AddDate(0, 0, -2).Format("2006-01-02")})
	a.Store.Create("task", map[string]any{"title": "Pay rent", "due": now.AddDate(0, 0, -1).Format("2006-01-02")})
	a.Store.Create("event", map[string]any{"title": "Dentist", "starts": now.AddDate(0, 0, -3).Format(time.RFC3339)})

	entries := get(t, h, "/t/entry").Body.String()
	if strings.Contains(entries, ">Overdue ") || !strings.Contains(entries, ">Earlier ") {
		t.Errorf("entries that happened are Earlier, not Overdue: %s", truncate(entries))
	}
	if regexp.MustCompile(`\(due [^)]*\)`).MatchString(entries) {
		t.Error("an entry is told apart by its time, not as due")
	}
	order := regexp.MustCompile(`datetime="([^"]+)"`).FindAllStringSubmatch(entries, -1)
	for i := 1; i < len(order); i++ {
		if order[i][1] > order[i-1][1] {
			t.Errorf("Earlier is the latest first: %v before %v", order[i-1][1], order[i][1])
		}
	}
	if regexp.MustCompile(`entries,</span></span></h2>`).MatchString(entries) {
		t.Error("no comma with nothing after it")
	}

	tasks := get(t, h, "/t/task").Body.String()
	if !strings.Contains(tasks, ">Overdue ") || !regexp.MustCompile(`\([A-Z][a-z]{2}\s+\d`).MatchString(tasks) {
		t.Errorf("a task past its day is Overdue, told apart by when it is due: %s", truncate(tasks))
	}
	if events := get(t, h, "/t/event").Body.String(); strings.Contains(events, ">Overdue ") || !strings.Contains(events, ">Earlier ") {
		t.Errorf("a meeting that has passed is Earlier: %s", truncate(events))
	}
}
