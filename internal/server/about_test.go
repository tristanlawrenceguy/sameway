package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Things know what they are about: a record's page offers a reminder
// about it, a word with the assistant about it, and its day on the
// calendar; the reminder rings with the thing and leads to it, and the
// thing's page lists it; a calendar can show everything with a day; a
// habit nudges through the clock when its time is past and it is not yet
// met; an entry is titled by its habit.
func TestThingsKnowWhatTheyAreAbout(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	task, err := a.Store.Create("task", map[string]any{"title": "Order compost", "due": when.Store(tomorrow, false)})
	if err != nil {
		t.Fatal(err)
	}
	cal, _ := a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "calendar", "props": map[string]any{"type": "all"}}))
	taskPath := "/t/task/" + task.ID

	// The task's page: the next things to do with it.
	page := get(t, h, taskPath).Body.String()
	for _, want := range []string{`aria-label="Do with this"`, `name="about" value="` + taskPath + `"`, `href="/chat?about=` + taskPath + `"`, `href="/canvas/` + cal.ID + `?day=` + tomorrow.Format("2006-01-02") + `"`, `>See that day<`} {
		if !strings.Contains(page, want) {
			t.Errorf("a record's page offers a reminder, the assistant and its day, missing %q\n%s", want, page)
		}
	}
	if chatPage := get(t, h, "/chat?about="+taskPath).Body.String(); !strings.Contains(chatPage, "About Order compost ("+taskPath+"): ") {
		t.Errorf("the chat opens with the thing in the box\n%s", chatPage)
	}

	// A reminder about the task, from its page: a timer with about set.
	req := httptest.NewRequest(http.MethodPost, "/clock/set", strings.NewReader(url.Values{"minutes": {"10"}, "about": {taskPath}, "title": {"Order compost"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "http://example.com"+taskPath)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	reminders, _ := a.Store.List(server.ReminderType, store.ListOptions{})
	if rec.Code != http.StatusSeeOther || len(reminders) != 1 || reminders[0].Fields["about"] != taskPath {
		t.Fatalf("a reminder set from a page is about it, got %d and %v", rec.Code, reminders)
	}
	page = get(t, h, taskPath).Body.String()
	if !strings.Contains(page, "Reminders about this") || !strings.Contains(page, `href="/t/reminder/`+reminders[0].ID+`"`) {
		t.Errorf("the thing's page lists the reminders about it\n%s", page)
	}
	if rp := get(t, h, "/t/reminder/"+reminders[0].ID).Body.String(); !strings.Contains(rp, `<dt>About</dt><dd data-prop="about" data-source="`+taskPath+`"><a class="sw-link" href="`+taskPath+`">Order compost</a></dd>`) {
		t.Errorf("a reminder's page leads to what it is about\n%s", rp)
	}

	// It rings with the thing and leads to it.
	var mu sync.Mutex
	var told []string
	done := make(chan struct{}, 4)
	srv.OnRing(func(title, text, url string) {
		mu.Lock()
		told = append(told, title+" | "+text+" | "+url)
		mu.Unlock()
		done <- struct{}{}
	})
	if rang := srv.Ring(now.Add(11 * time.Minute)); len(rang) != 1 {
		t.Fatalf("the timer rings when its time comes, got %v", rang)
	}
	<-done
	mu.Lock()
	if len(told) != 1 || told[0] != "Order compost | Order compost | "+taskPath {
		t.Errorf("a ring beyond the page says what it is about and leads there, got %v", told)
	}
	told = nil
	mu.Unlock()

	// The calendar shows everything with a day, each saying what it is.
	day := get(t, h, "/canvas/"+cal.ID+"?day="+tomorrow.Format("2006-01-02")).Body.String()
	if !strings.Contains(day, `href="`+taskPath+`"`) || !strings.Contains(day, `<span class="sw-calendar__meta">task</span>`) {
		t.Errorf("a calendar of type all has the task on its day, saying it is a task\n%s", day)
	}
	today := get(t, h, "/canvas/"+cal.ID+"?day="+now.Format("2006-01-02")).Body.String()
	if !strings.Contains(today, `href="/t/reminder/`+reminders[0].ID+`"`) || !strings.Contains(today, `<span class="sw-calendar__meta">reminder</span>`) {
		t.Errorf("and the reminder on its day\n%s", today)
	}

	// A habit with a remind time, not met once the time is past, rings
	// once a day as a reminder about it, with where it stands; one that
	// is met does not.
	water, _ := a.Store.Create(server.HabitType, map[string]any{"name": "Water", "target": 8, "unit": "glasses", "remind": "00:00"})
	stretch, _ := a.Store.Create(server.HabitType, map[string]any{"name": "Stretch", "remind": "00:00"})
	a.Store.Create(server.EntryType, map[string]any{"habit": stretch.ID, "at": when.Store(now, false)})
	entry, _ := a.Store.Create(server.EntryType, map[string]any{"habit": water.ID, "at": when.Store(now, false), "amount": 3})
	rang := srv.Ring(now)
	<-done
	mu.Lock()
	if len(rang) != 1 || rang[0].Fields["about"] != "/t/habit/"+water.ID || len(told) != 1 || told[0] != "Water | Water: 3 of 8 glasses so far | /t/habit/"+water.ID {
		t.Errorf("the unmet habit nudges with where it stands, the met one does not: %v %v", rang, told)
	}
	mu.Unlock()
	if again := srv.Ring(now.Add(time.Hour)); len(again) != 0 {
		t.Errorf("a habit nudges once a day, got %v", again)
	}
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `data-component="clock"`) && !strings.Contains(page, "Water") {
		t.Log("no clock on the canvas; the nudge still rings on the stream and the habit page")
	}

	// An entry is titled by its habit and how much.
	if ep := get(t, h, "/t/entry/"+entry.ID).Body.String(); !strings.Contains(ep, "<h1") || !strings.Contains(ep, "Water: 3 glasses") {
		t.Errorf("an entry reads as its habit and amount\n%s", ep)
	}
}
