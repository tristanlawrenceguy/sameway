package server_test

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A repeat is written in words, kept as a rule and said back as it was
// read, on the page and when it is saved; words that are not a repeat are
// refused with what they must be.
func TestARepeatIsWrittenInWordsAndSaidBack(t *testing.T) {
	a, h := newApp(t)
	task, _ := a.Store.Create("task", map[string]any{"title": "Water the ferns", "due": "2026-10-06T00:00:00Z"})

	page := after(t, h, postForm(t, h, "/t/task/"+task.ID+"/props", url.Values{"prop-repeat": {"every tues"}})).Body.String()
	if !strings.Contains(page, "Repeat is every Tuesday.") {
		t.Errorf("saving a repeat says how it was read\n%s", truncate(page))
	}
	if rec, _ := a.Store.Get("task", task.ID); rec.Fields["repeat"] != "FREQ=WEEKLY;BYDAY=TU" {
		t.Errorf("kept as a rule, got %v", rec.Fields["repeat"])
	}
	if page := get(t, h, "/t/task/"+task.ID+fieldsView).Body.String(); !strings.Contains(page, "Every Tuesday") || !strings.Contains(page, `data-kind="repeat" data-source="every Tuesday"`) {
		t.Errorf("the page says it in words and the editor edits the words\n%s", truncate(page))
	}

	page = after(t, h, postForm(t, h, "/t/task/"+task.ID+"/props", url.Values{"prop-repeat": {"now and then"}})).Body.String()
	if !strings.Contains(page, "Repeat must say how often, like every day") {
		t.Errorf("words that are not a repeat are refused with what it must be\n%s", truncate(page))
	}

	// The field says it back as it is typed, from the same reading.
	var read struct{ Text, Rule, Error string }
	decode(t, get(t, h, "/when?repeat="+url.QueryEscape("every month on the 31st")), &read)
	if read.Text != "every month on the 31st" || read.Rule != "FREQ=MONTHLY;BYMONTHDAY=31" {
		t.Errorf("GET /when reads a repeat, got %+v", read)
	}
	res := get(t, h, "/when?repeat="+url.QueryEscape("every month on the 32nd"))
	wantStatus(t, res, http.StatusUnprocessableEntity)
	if decode(t, res, &read); !strings.Contains(read.Error, "1st to the 31st") {
		t.Errorf("and says what it must be, got %+v", read)
	}
}

// A repeating task ticked done is due again on its next day, counted from
// the day it was due, in one change the log has and Undo takes back.
func TestARepeatingTaskDoneIsDueAgain(t *testing.T) {
	a, h := newApp(t)
	due := time.Now().AddDate(0, 0, 1).Format("2006-01-02") + "T00:00:00Z"
	task, _ := a.Store.Create("task", map[string]any{"title": "Water the ferns", "due": due, "repeat": "every 2 weeks"})

	page := after(t, h, postForm(t, h, "/t/task/"+task.ID+"/props", url.Values{"prop-done": {"true", "false"}})).Body.String()
	rec, _ := a.Store.Get("task", task.ID)
	want, _ := when.Next("FREQ=WEEKLY;INTERVAL=2", due, time.Now())
	if rec.Fields["done"] != false || rec.Fields["due"] != want {
		t.Errorf("done, it is due again two weeks after it was due (%s), got %v %v", want, rec.Fields["done"], rec.Fields["due"])
	}
	if !strings.Contains(page, "Water the ferns is done.") || !strings.Contains(page, "It repeats every 2 weeks, so it is due again "+when.Text(want)+".") {
		t.Errorf("ticking says it is done and when it is due again\n%s", truncate(page))
	}
	undo := regexp.MustCompile(`action="(/activity/[^"]+/undo)"`).FindStringSubmatch(page)
	if undo == nil {
		t.Fatalf("it can be undone\n%s", truncate(page))
	}
	after(t, h, postForm(t, h, undo[1], url.Values{"from": {"/"}}))
	if back, _ := a.Store.Get("task", task.ID); back.Fields["due"] != due || back.Fields["done"] != false {
		t.Errorf("undo puts back the day it was due, got %v %v", back.Fields["due"], back.Fields["done"])
	}

	// Through the API too, and a monthly one keeps its day after February.
	monthly, _ := a.Store.Create("task", map[string]any{"title": "Pay rent", "due": "2027-01-31", "repeat": "every month"})
	wantStatus(t, postJSON(t, h, http.MethodPatch, "/api/task/"+monthly.ID, map[string]any{"done": true}), http.StatusOK)
	rec, _ = a.Store.Get("task", monthly.ID)
	if rec.Fields["due"] != "2027-02-28T00:00:00Z" || rec.Fields["repeat"] != "FREQ=MONTHLY;BYMONTHDAY=31" {
		t.Errorf("the 31st falls on the 28th in February and stays the 31st, got %v %v", rec.Fields["due"], rec.Fields["repeat"])
	}

	// A repeat that has ended leaves it done.
	ended, _ := a.Store.Create("task", map[string]any{"title": "Last lesson", "due": "2026-01-10", "repeat": "FREQ=DAILY;UNTIL=20260110"})
	a.Store.Update("task", ended.ID, map[string]any{"done": true})
	if rec, _ := a.Store.Get("task", ended.ID); rec.Fields["done"] != true {
		t.Errorf("past its end it stays done, got %v", rec.Fields["done"])
	}
}

// A repeating reminder dismissed is set for its next time at the same time
// of day, even after five more minutes; the clock says that it repeats, in
// words, and Cancel on one that repeats skips only this time.
func TestARepeatingReminderRingsAgain(t *testing.T) {
	a, h := newApp(t)
	a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "clock", "props": map[string]any{}}))
	now := time.Now()
	at := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, time.Local).Add(-time.Minute)
	rem, _ := a.Store.Create(server.ReminderType, map[string]any{"title": "Stretch", "at": when.Store(at, false), "state": "rang", "repeat": "every day"})

	if page := get(t, h, "/").Body.String(); !strings.Contains(page, "Repeats every day") {
		t.Errorf("a ringing reminder says it repeats\n%s", truncate(page))
	}
	after(t, h, postForm(t, h, "/clock/"+rem.ID+"/snooze", nil))
	page := after(t, h, postForm(t, h, "/clock/"+rem.ID+"/done", nil)).Body.String()
	rec, _ := a.Store.Get(server.ReminderType, rem.ID)
	next := when.Store(at.AddDate(0, 0, 1), false)
	if rec.Fields["state"] != "set" || rec.Fields["at"] != next || rec.Fields["repeat"] != "FREQ=DAILY" {
		t.Errorf("dismissed, it is set for tomorrow at its own time (%s), got %v", next, rec.Fields)
	}
	if !strings.Contains(page, "Stretch: this time skipped") || !strings.Contains(page, "It repeats every day, so it rings again tomorrow at "+when.Clock(at)+".") {
		t.Errorf("dismissing says when it rings again\n%s", said(page))
	}
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, "Skip this time") || !strings.Contains(page, "Repeats every day") {
		t.Errorf("coming up says it repeats and offers to skip this time\n%s", truncate(page))
	}
	page = after(t, h, postForm(t, h, "/clock/"+rem.ID+"/done", nil)).Body.String()
	if rec, _ := a.Store.Get(server.ReminderType, rem.ID); rec.Fields["at"] != when.Store(at.AddDate(0, 0, 2), false) || !strings.Contains(page, "Stretch: this time skipped") {
		t.Errorf("skipping sets the time after, got %v\n%s", rec.Fields["at"], truncate(page))
	}

	// Dismissed while it rings, it says so, with its next time.
	water, _ := a.Store.Create(server.ReminderType, map[string]any{"title": "Drink water", "at": when.Store(at, false), "state": "rang", "repeat": "every 2 days"})
	page = after(t, h, postForm(t, h, "/clock/"+water.ID+"/done", nil)).Body.String()
	if !strings.Contains(page, "Drink water dismissed") || !strings.Contains(page, "It repeats every 2 days, so it rings again on ") {
		t.Errorf("dismissing a ringing one says when it rings again\n%s", said(page))
	}
}

// A workspace made before repeats gets the field on its tasks and
// reminders, as it gets any field of a type Sameway provides.
func TestAnOlderWorkspaceGetsRepeat(t *testing.T) {
	dir := t.TempDir()
	known := filepath.Join(t.TempDir(), "known.json")
	os.WriteFile(known, []byte("[]"), 0o644)
	t.Setenv("SAMEWAY_KNOWN", known)
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	old := "name: task\ntitle: title\nprovided: true\nfields:\n  title:\n    type: string\n  done:\n    type: bool\n  due:\n    type: datetime\n"
	if err := os.WriteFile(filepath.Join(dir, "schema", "task.yaml"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	for _, name := range []string{"task", server.ReminderType} {
		typ, _ := a.Types.Get(name)
		if f, ok := typ.Field("repeat"); !ok || f.Type != "repeat" {
			t.Errorf("an older %s gets repeat", name)
		}
	}
}
