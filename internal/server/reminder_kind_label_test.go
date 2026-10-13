package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A reminder's kind field is called Type, not Kind: the detail page for an
// alarm and a timer both show <dt>Type</dt>, their values render as "Alarm"
// and "Timer", and no other record type's labels change.
func TestReminderKindIsCalledType(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 7, 23, 45, 0, 0, time.Local)
	a, h := newAppWith(t, app.Options{Clock: func() time.Time { return now }})

	alarm, _ := a.Store.Create(server.ReminderType, map[string]any{
		"title": "Wake up", "at": when.Store(now.AddDate(0, 0, 1), false), "kind": "alarm",
	})
	timer, _ := a.Store.Create(server.ReminderType, map[string]any{
		"title": "Boil the kettle", "at": when.Store(now.AddDate(0, 0, 2), false), "kind": "timer",
	})

	// Acceptance 1: opening /t/reminder/<id> shows <dt>Type</dt>, not <dt>Kind</dt>.
	for _, id := range []string{alarm.ID, timer.ID} {
		page := get(t, h, "/t/reminder/"+id).Body.String()
		if strings.Contains(page, `<dt>Kind</dt>`) {
			t.Errorf("reminder %s should not show <dt>Kind</dt>\n%s", id, truncate(page))
		}
		if !strings.Contains(page, `<dt>Type</dt>`) {
			t.Errorf("reminder %s should show <dt>Type</dt>, got:\n%s", id, truncate(page))
		}
	}

	// Acceptance 2: enum values render as display names, not raw keys.
	alarmPage := get(t, h, "/t/reminder/"+alarm.ID).Body.String()
	if !strings.Contains(alarmPage, "Alarm") {
		t.Errorf("the alarm page should display \"Alarm\" as the kind value\n%s", truncate(alarmPage))
	}

	timerPage := get(t, h, "/t/reminder/"+timer.ID).Body.String()
	if !strings.Contains(timerPage, "Timer") {
		t.Errorf("the timer page should display \"Timer\" as the kind value\n%s", truncate(timerPage))
	}

	// Acceptance 3: other record types are unaffected.
	taskPage := get(t, h, "/t/task").Body.String()
	if strings.Contains(taskPage, `<dt>Type</dt>`) {
		t.Errorf("the task list page should not show <dt>Type</dt>, labels unchanged\n%s", truncate(taskPage))
	}

	entryPage := get(t, h, "/t/entry").Body.String()
	if strings.Contains(entryPage, `<dt>Type</dt>`) {
		t.Errorf("the entry list page should not show <dt>Type</dt>, labels unchanged\n%s", truncate(entryPage))
	}

	projectPage := get(t, h, "/t/project").Body.String()
	if strings.Contains(projectPage, `<dt>Type</dt>`) {
		t.Errorf("the project list page should not show <dt>Type</dt>, labels unchanged\n%s", truncate(projectPage))
	}
}
