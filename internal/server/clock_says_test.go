package server_test

import (
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Setting a reminder says back what the server understood, and when it
// rings: the time was typed the way a person says it, so a wrong reading
// is caught here, not when the alarm fails to ring. Putting one off says
// when it rings again, and each can be undone from the message. The app's
// clock is fixed at a quarter to midnight: a timer set then rings
// tomorrow, and says so, which once failed this test only near midnight.
func TestTheClockSaysWhatItSet(t *testing.T) {
	now := time.Date(2026, 10, 7, 23, 45, 0, 0, time.Local)
	a, h := newAppWith(t, app.Options{Clock: func() time.Time { return now }})
	page := after(t, h, postForm(t, h, "/clock/set", url.Values{"at": {"tomorrow 7am"}, "title": {"Call the vet"}})).Body.String()
	if !strings.Contains(page, "Alarm set") || !strings.Contains(page, "Call the vet rings tomorrow at 7am.") {
		t.Errorf("setting an alarm says what and when\n%s", truncate(page))
	}
	page = after(t, h, postForm(t, h, "/clock/set", url.Values{"minutes": {"30"}})).Body.String()
	if !strings.Contains(page, "Timer set") || !strings.Contains(page, "30 minute timer rings tomorrow at 12:15am.") {
		t.Errorf("setting a timer says when it rings, tomorrow at 12:15am\n%s", truncate(page))
	}

	due, _ := a.Store.Create(server.ReminderType, map[string]any{"title": "Tea", "at": when.Store(now.Add(-time.Minute), false), "state": "rang"})
	page = after(t, h, postForm(t, h, "/clock/"+due.ID+"/snooze", nil)).Body.String()
	if !strings.Contains(page, "Tea: 5 more minutes") || !strings.Contains(page, "Rings again at ") {
		t.Errorf("five more minutes says when it rings again\n%s", truncate(page))
	}
	undo := regexp.MustCompile(`action="(/activity/[^"]+/undo)"`).FindStringSubmatch(page)
	if undo == nil {
		t.Fatalf("putting it off can be undone\n%s", truncate(page))
	}
	after(t, h, postForm(t, h, undo[1], url.Values{"from": {"/"}}))
	if back, _ := a.Store.Get(server.ReminderType, due.ID); back.Fields["state"] != "rang" {
		t.Errorf("undo rings it again, got %v", back.Fields["state"])
	}
	page = after(t, h, postForm(t, h, "/clock/"+due.ID+"/done", nil)).Body.String()
	if !strings.Contains(page, "Tea dismissed") {
		t.Errorf("dismissing says so\n%s", truncate(page))
	}
}

// Coming up is one list in time order: a reminder and a task at the same
// part of the day sit side by side whatever they are, and a reminder on
// another day has its day above its time.
func TestComingUpIsOneListInTimeOrder(t *testing.T) {
	a, h := newApp(t)
	if _, err := a.Store.Create(records.BlockType, a.Chat.BlockFields(map[string]any{"component": "clock", "props": map[string]any{}})); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	day := func(d, hour int) time.Time {
		return time.Date(now.Year(), now.Month(), now.Day()+d, hour, 0, 0, 0, time.Local)
	}
	a.Store.Create(server.ReminderType, map[string]any{"title": "Call the vet", "at": when.Store(day(1, 9), false), "state": "set", "kind": "alarm"})
	a.Store.Create("task", map[string]any{"title": "Water the tomatoes", "due": when.Store(day(0, 23), false)})
	page := get(t, h, "/").Body.String()
	task, vet := strings.Index(page, "Water the tomatoes"), strings.Index(page, "Call the vet")
	if task < 0 || vet < 0 || task > vet {
		t.Errorf("today's task comes before tomorrow's alarm\n%s", truncate(page))
	}
	if !strings.Contains(page, `<span class="sw-clock__when"><span class="sw-clock__day">Tomorrow</span> 9am</span>`) {
		t.Errorf("a reminder on another day has its day above its time\n%s", truncate(page))
	}
}
