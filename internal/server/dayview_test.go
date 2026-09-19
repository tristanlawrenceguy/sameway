package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A calendar shows one day when asked: from the month, each day number
// leads to its day on the block's own page, where what is on that day is
// laid out by the hour, all-day things first, with the days either side
// and the month a link away.
func TestACalendarShowsOneDayByTheHour(t *testing.T) {
	a, h := newApp(t)
	blk, err := a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "calendar", "props": map[string]any{"type": "task"}}))
	if err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 9, 21, 0, 0, 0, 0, time.Local)
	a.Store.Create("task", map[string]any{"title": "Plant garlic", "due": when.Store(day, true)})
	a.Store.Create("task", map[string]any{"title": "Call the vet", "due": when.Store(day.Add(9*time.Hour+30*time.Minute), false)})
	a.Store.Create("task", map[string]any{"title": "Late watering", "due": when.Store(day.Add(21*time.Hour), false)})

	month := get(t, h, "/canvas/"+blk.ID+"?month=2026-09").Body.String()
	if !strings.Contains(month, `href="/canvas/`+blk.ID+`?day=2026-09-21"`) {
		t.Errorf("each day number of the month leads to its day\n%s", month)
	}

	page := get(t, h, "/canvas/"+blk.ID+"?day=2026-09-21").Body.String()
	if !strings.Contains(page, `data-day="2026-09-21"`) || !strings.Contains(page, "Monday 21 September 2026") {
		t.Fatalf("the day view names the day\n%s", page)
	}
	if !strings.Contains(page, `aria-label="All day"`) || !strings.Contains(page, "Plant garlic") {
		t.Error("a task with a day and no time is all day")
	}
	hours := page[strings.Index(page, `sw-calendar__hours"`):]
	if !strings.Contains(hours, ">09:00<") || strings.Index(hours, ">09:00<") > strings.Index(hours, "Call the vet") {
		t.Error("a timed task sits in its hour")
	}
	if !strings.Contains(hours, ">21:00<") || strings.Contains(hours, ">23:00<") {
		t.Error("the hours reach as late as the latest thing and no further")
	}
	if !strings.Contains(page, `?day=2026-09-20"`) || !strings.Contains(page, `?day=2026-09-22"`) || !strings.Contains(page, `?month=2026-09">September<`) {
		t.Errorf("the days either side and the month are a link away\n%s", page)
	}
	if !strings.Contains(get(t, h, "/canvas/"+blk.ID+"?day=2026-09-22").Body.String(), "Nothing on this day.") {
		t.Error("an empty day says so")
	}
}
