package server_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A record says the same at a glance wherever it is met: its row in its
// list, a list on the canvas, its own page. Three functions once said it
// three ways, and the same task read "Due Fri 9 Oct 2026, 14:00" on the
// canvas and "In 4 days at 2:00pm" in its list.
func TestARecordSaysTheSameEverywhere(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	ana, _ := a.Store.Create("person", map[string]any{"name": "Ana Silva"})
	due := time.Now().AddDate(0, 0, 3).Format("2006-01-02") + " 14:00"
	task, _ := a.Store.Create("task", map[string]any{"title": "Buy paint", "status": "doing", "due": due, "for": ana.ID})
	a.Store.Create(records.BlockType, a.Chat.BlockFields(map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Tasks"}}))
	day := when.Relative(task.Fields["due"].(string), time.Now(), false)

	pages := map[string]string{
		"its row":    read(get(t, h, "/t/task").Body.String()),
		"the canvas": read(get(t, h, "/").Body.String()),
		"its page":   read(get(t, h, "/t/task/"+task.ID).Body.String()),
	}
	for where, page := range pages {
		for _, says := range []string{"Doing", day, "Ana Silva"} {
			if !strings.Contains(page, says) {
				t.Errorf("%s does not say %q", where, says)
			}
		}
		if strings.Contains(page, "In 3 days") {
			t.Errorf("%s makes a person count days", where)
		}
	}
	// Its page and the canvas name the day by its field; its row is in a
	// list of days due, under a heading that says when.
	for _, where := range []string{"the canvas", "its page"} {
		if !strings.Contains(pages[where], "Due "+day) {
			t.Errorf("%s does not name the day by its field: want %q", where, "Due "+day)
		}
	}
	// And a done one says nothing of its state but its tick.
	postJSON(t, h, http.MethodPatch, "/api/task/"+task.ID, map[string]any{"status": "done"})
	if page := get(t, h, "/t/task").Body.String(); strings.Contains(page, ">Done</span>") && strings.Contains(page, "checked") {
		t.Error("a ticked task's row says Done once, by its box")
	}
}

// Late is said in words, not in amber alone, and only of what can be
// done: a meeting that happened yesterday is past, not overdue.
func TestLateIsSaidInWordsAndOnlyOfWhatCanBeDone(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	task, _ := a.Store.Create("task", map[string]any{"title": "Pay rent", "due": yesterday})
	meeting, _ := a.Store.Create("event", map[string]any{"title": "Standup", "starts": yesterday + " 09:00"})
	a.Store.Create(records.BlockType, a.Chat.BlockFields(map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Tasks"}}))

	for where, page := range map[string]string{"its page": get(t, h, "/t/task/"+task.ID).Body.String(), "the canvas": get(t, h, "/").Body.String()} {
		if !strings.Contains(read(page), "Overdue, due yesterday") {
			t.Errorf("%s does not say the task is late in words", where)
		}
	}
	if page := get(t, h, "/t/event/"+meeting.ID).Body.String(); strings.Contains(page, "Overdue") || strings.Contains(page, "sw-badge--warning") {
		t.Error("a meeting that happened is said as late")
	}
	// Ticked, it is not late any more.
	postJSON(t, h, http.MethodPatch, "/api/task/"+task.ID, map[string]any{"done": true})
	if strings.Contains(read(get(t, h, "/t/task/"+task.ID).Body.String()), "Overdue") {
		t.Error("a done task is said as late")
	}
}

// Every day a glance, a field or a made-line says sits in a <time> that
// holds its value for a machine; the words leave the date out only with
// the date in full beside them for a pointer.
func TestDaysAreHeldInTimeElements(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	today := time.Now().Format("2006-01-02")
	task, _ := a.Store.Create("task", map[string]any{"title": "Water plants", "due": today})
	row := get(t, h, "/t/task").Body.String()
	if !strings.Contains(row, `<time class="sw-when sw-when--today" datetime="`+today+`" title="`) {
		t.Errorf("the row's day is not a time with its value and the date in full:\n%s", around(row, "sw-when"))
	}
	page := get(t, h, "/t/task/"+task.ID).Body.String()
	if !strings.Contains(page, `<time datetime="`+today+`"`) {
		t.Error("the lede's day is not held in a time element")
	}
	if !strings.Contains(page, `class="sw-detail__when sw-muted sw-small"><time datetime="`) {
		t.Error("when it was made is not held in a time element")
	}
	postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Seeds"})
	if !strings.Contains(get(t, h, "/t/note").Body.String(), `<time class="sw-muted" datetime="`) {
		t.Error("a row's last change is not held in a time element")
	}
}

// Each provided type says what its rules give it, and nothing a type's
// name was patched in for: the table in design/foundations/glance.md.
func TestEveryProvidedTypeSaysItsGlance(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	rec := func(typ string, fields map[string]any) string {
		res := postJSON(t, h, http.MethodPost, "/api/"+typ, fields)
		var out struct{ ID string }
		json.Unmarshal(res.Body.Bytes(), &out)
		if out.ID == "" {
			t.Fatalf("making a %s: %s", typ, res.Body.String())
		}
		res = get(t, h, "/api/"+typ+"/"+out.ID)
		var view struct{ Glance string }
		json.Unmarshal(res.Body.Bytes(), &view)
		return view.Glance
	}
	cases := []struct {
		typ    string
		fields map[string]any
		want   string
	}{
		{"task", map[string]any{"title": "Buy paint", "status": "doing", "due": tomorrow + " 14:00"}, "Doing · Due tomorrow at " + when.Clock(time.Date(1, 1, 1, 14, 0, 0, 0, time.UTC), false)},
		{"note", map[string]any{"title": "Seeds", "status": "published", "pinned": true}, "Published · Pinned"},
		{"event", map[string]any{"title": "Standup", "starts": tomorrow}, "Starts tomorrow"},
		{"reminder", map[string]any{"title": "Call", "at": tomorrow + " 07:00"}, "Tomorrow at " + when.Clock(time.Date(1, 1, 1, 7, 0, 0, 0, time.UTC), false)},
		{"project", map[string]any{"title": "House", "status": "done"}, "Done"},
		{"project", map[string]any{"title": "Garden"}, ""},
		{"person", map[string]any{"name": "Ana Silva"}, ""},
	}
	for _, c := range cases {
		if got := rec(c.typ, c.fields); got != c.want {
			t.Errorf("a %s says %q at a glance, want %q", c.typ, got, c.want)
		}
	}
	_ = a
}

// The clock is the person's: 12-hour the GOV.UK way unless they choose 24.
func TestTimesFollowThePersonsClock(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	a.Store.Create("task", map[string]any{"title": "Dentist", "due": tomorrow + " 14:30"})
	if page := read(get(t, h, "/t/task").Body.String()); !strings.Contains(page, "Tomorrow at 2:30pm") {
		t.Errorf("the 12-hour clock is not the default for English:\n%s", page)
	}
	postForm(t, h, "/help/set", map[string][]string{"key": {"ui.clock"}, "value": {"24"}})
	defer a.Workspace.Set("ui.clock", "")
	if page := read(get(t, h, "/t/task").Body.String()); !strings.Contains(page, "Tomorrow at 14:30") {
		t.Errorf("the 24-hour clock chosen is not used:\n%s", page)
	}
	if !strings.Contains(get(t, h, "/t/task").Body.String(), ` data-clock="24"`) {
		t.Error("the page does not tell its scripts the clock")
	}
}

// Two workspaces open in one program each say times on their own clock:
// the choice was once kept on the program, and the last workspace opened
// set it for both.
func TestTwoWorkspacesKeepTheirOwnClocks(t *testing.T) {
	t.Parallel()
	a12, h12 := newApp(t)
	a24, h24 := newApp(t)
	if err := a24.Workspace.Set("ui.clock", "24"); err != nil {
		t.Fatal(err)
	}
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	for _, a := range []*app.App{a12, a24} {
		a.Store.Create("task", map[string]any{"title": "Dentist", "due": tomorrow + " 14:30"})
	}
	if page := read(get(t, h12, "/t/task").Body.String()); !strings.Contains(page, "Tomorrow at 2:30pm") {
		t.Errorf("the other workspace's 24-hour clock is used here:\n%s", page)
	}
	if page := read(get(t, h24, "/t/task").Body.String()); !strings.Contains(page, "Tomorrow at 14:30") {
		t.Errorf("the 24-hour clock chosen is not used:\n%s", page)
	}
}

// A page tells its scripts this computer's zone and today, so its days
// stay true while it is open and a reader elsewhere is told the zone.
func TestAPageSaysItsZoneAndDay(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	page := get(t, h, "/t/task").Body.String()
	if !strings.Contains(page, ` data-today="`+time.Now().Format("2006-01-02")+`"`) || !strings.Contains(page, ` data-zone="`) {
		t.Errorf("the root element does not say the zone and today: %s", around(page, "<html"))
	}
}

// read is what a person reads on a page (machine_words_test.go).
func read(page string) string { return strings.Join(peopleText(page), "\n") }

// around is a page near where it says what.
func around(page, what string) string {
	i := strings.Index(page, what)
	if i < 0 {
		return "(not there)"
	}
	return page[max(0, i-200):min(len(page), i+300)]
}
