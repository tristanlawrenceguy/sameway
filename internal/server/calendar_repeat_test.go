package server_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A thing that repeats is on the calendar on each day it falls in the
// month shown, each saying in words that it repeats, once; only the one
// due now carries its tick, and the month shown is all that is worked out.
func TestACalendarShowsARepeatOnEachDay(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	ferns, _ := a.Store.Create("task", map[string]any{"title": "Water the ferns", "due": "2026-09-01", "repeat": "every Tuesday"})
	a.Store.Create(server.ReminderType, map[string]any{"title": "Stretch", "at": "2026-09-28T07:00:00Z", "repeat": "every day until 30 Sep 2026"})
	var block struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "calendar", "props": map[string]any{"type": "task", "month": "2026-09", "detail": "page"},
	}), &block)

	page := get(t, h, "/canvas/"+block.ID).Body.String()
	grid := regexp.MustCompile(`(?s)<table.*?</table>`).FindString(page)
	for _, day := range []string{"2026-09-01", "2026-09-08", "2026-09-15", "2026-09-22", "2026-09-29"} {
		cell := regexp.MustCompile(`(?s)<td class="sw-calendar__day[^"]*" data-date="` + day + `".*?</td>`).FindString(grid)
		if !strings.Contains(cell, "Water the ferns") || strings.Count(cell, "Repeats every Tuesday") != 1 {
			t.Errorf("on %s the task shows, saying once that it repeats\n%s", day, cell)
		}
	}
	if n := strings.Count(grid, `data-field="done"`); n != 1 {
		t.Errorf("only the one due now carries its tick, got %d", n)
	}
	if !strings.Contains(grid, `href="/t/task/`+ferns.ID+`"`) {
		t.Error("each day leads to the task")
	}

	// The next month shows that month's Tuesdays, and no further.
	page = get(t, h, "/canvas/"+block.ID+"?month=2026-10").Body.String()
	grid = regexp.MustCompile(`(?s)<table.*?</table>`).FindString(page)
	if n := strings.Count(grid, "Repeats every Tuesday"); n != 4 {
		t.Errorf("October has four Tuesdays, got %d", n)
	}

	// Everything together: the reminder's days to its end, each said.
	decode(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "calendar", "props": map[string]any{"type": "all", "month": "2026-09"},
	}), &block)
	grid = regexp.MustCompile(`(?s)<table.*?</table>`).FindString(get(t, h, "/canvas/"+block.ID).Body.String())
	if n := strings.Count(grid, "Repeats every day until Wed 30 Sep 2026"); n != 3 {
		t.Errorf("a daily reminder from the 28th to the 30th is on three days, got %d", n)
	}
}
