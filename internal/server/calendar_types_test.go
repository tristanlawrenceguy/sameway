package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// typesMonth is kindsMonth with a habit logged every few days, the entries
// that flooded a calendar of everything in the agent evaluation, a
// reminder of one name with a task on its day, and an event.
func typesMonth(t *testing.T, h http.Handler, props map[string]any) string {
	t.Helper()
	habit := postJSON(t, h, http.MethodPost, "/api/"+server.HabitType, map[string]any{"name": "Drink water", "unit": "glasses", "target": 8})
	wantStatus(t, habit, http.StatusCreated)
	var made struct{ ID string }
	decode(t, habit, &made)
	for _, day := range []string{"2026-09-01", "2026-09-05", "2026-09-09"} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/"+server.EntryType, map[string]any{"habit": made.ID, "at": day + "T00:00:00Z", "amount": 6}), http.StatusCreated)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{"title": "Plant garlic", "at": "2026-09-22T00:00:00Z"}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/event", map[string]any{"title": "Harvest fair", "starts": "2026-09-12T00:00:00Z"}), http.StatusCreated)
	return kindsMonth(t, h, props)
}

// A calendar of tasks and reminders shows those two kinds and nothing
// else, offers the two to narrow to, and says no kind in its events' text.
func TestACalendarShowsTheTypesItNames(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	id := typesMonth(t, h, map[string]any{"types": []any{"task", "reminder"}, "month": "2026-09", "detail": "page"})
	own := "/canvas/" + id
	page := get(t, h, own).Body.String()
	for _, want := range []string{
		"Order compost", "Call the vet",
		`href="` + own + `" aria-current="page">All (4)</a>`,
		`?c-` + id + `-type=reminder">Reminders (2)</a>`,
		`?c-` + id + `-type=task">Tasks (2)</a>`,
		`data-kind="task"`, `data-kind="reminder"`,
		// Two of one name on one day are told apart by their kind alone.
		`Plant garlic<span class="sw-visually-hidden"> (task)</span>`,
		`Plant garlic<span class="sw-visually-hidden"> (reminder)</span>`,
		`href="/export/all.ics?type=task&amp;type=reminder"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("a calendar of tasks and reminders: %s\n%s", want, truncate(page))
		}
	}
	for _, not := range []string{"Drink water", "Harvest fair", "Entries (", "Events (", `sw-calendar__meta">task`, `sw-calendar__meta">reminder`} {
		if strings.Contains(page, not) {
			t.Errorf("only the kinds named, and no kind word after an event: %s", not)
		}
	}

	// The next month keeps the kinds, which are the block's own.
	next := get(t, h, own+"?month=2026-10").Body.String()
	if !strings.Contains(next, ">Harvest</a>") || strings.Contains(next, "Drink water") {
		t.Errorf("the months either side keep the types\n%s", truncate(next))
	}
	// Narrowed to one, the months either side keep it and it goes out alone.
	tasks := get(t, h, own+"?c-"+id+"-type=task").Body.String()
	for _, want := range []string{
		`href="` + own + `?c-` + id + `-type=task&amp;month=2026-10" rel="next"`,
		`aria-current="page">Tasks (2)</a>`,
		`href="/export/all.ics?type=task"`,
	} {
		if !strings.Contains(tasks, want) {
			t.Errorf("narrowed to tasks: %s\n%s", want, truncate(tasks))
		}
	}
	if strings.Contains(tasks, "Call the vet") {
		t.Error("a reminder is not a task")
	}

	ics := get(t, h, "/export/all.ics?type=task&type=reminder").Body.String()
	for _, want := range []string{"SUMMARY:Order compost", "SUMMARY:Call the vet"} {
		if !strings.Contains(ics, want) {
			t.Errorf("the calendar goes out with its kinds: %s\n%s", want, ics)
		}
	}
	if strings.Contains(ics, "Harvest fair") {
		t.Error("and without the kinds it does not show")
	}
}

// type all still shows every listed type, entries included.
func TestACalendarOfEverythingStillShowsAll(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	id := typesMonth(t, h, map[string]any{"type": "all", "month": "2026-09", "detail": "page"})
	page := get(t, h, "/canvas/"+id).Body.String()
	for _, want := range []string{"Drink water", "Harvest fair", "Order compost", "Call the vet", `href="/export/all.ics"`} {
		if !strings.Contains(page, want) {
			t.Errorf("a calendar of everything shows %s", want)
		}
	}
	if strings.Contains(page, `sw-calendar__meta">entry`) {
		t.Error("no kind word after an event")
	}
}

// Types are checked when written: one that is not there, or has no day,
// is refused with the page's reason; the rest say what they show.
func TestACalendarsTypesAreCheckedWhenWritten(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	res := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "calendar", "props": map[string]any{"types": []any{"task", "tasks", "reminder"}}})
	wantStatus(t, res, http.StatusUnprocessableEntity)
	if !strings.Contains(res.Body.String(), "there is no content type tasks") {
		t.Errorf("a type not there is said: %s", res.Body.String())
	}
	res = postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "calendar", "props": map[string]any{"types": []any{"task", "note"}}})
	wantStatus(t, res, http.StatusUnprocessableEntity)
	if !strings.Contains(res.Body.String(), "note has no date field") {
		t.Errorf("a type with no day is said: %s", res.Body.String())
	}
	var made struct{ Shows string }
	res = postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "calendar", "props": map[string]any{"types": []any{"task", "reminder"}}})
	wantStatus(t, res, http.StatusCreated)
	decode(t, res, &made)
	if made.Shows != "tasks and reminders with a date, 0 in all" {
		t.Errorf("a calendar of several says which, got %q", made.Shows)
	}
}
