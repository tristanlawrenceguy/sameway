package agenda_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A whole day is kept as midnight UTC. Today reads it as that day here,
// never as 2am, nor as yesterday west of Greenwich; a late task says the
// day it was due, a timed one its day and time; each day is in a <time>
// with its value; and Today is in the navigation.
func TestTodaySaysEachDayAsAPersonPlansBy(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	now := time.Now()
	midnight := func(n int) string { return now.AddDate(0, 0, n).Format("2006-01-02") + "T00:00:00Z" }
	y := now.AddDate(0, 0, -1)
	a.Store.Create("task", map[string]any{"title": "Pay rent", "due": midnight(0)})
	a.Store.Create("task", map[string]any{"title": "Renew passport", "due": midnight(-3)})
	a.Store.Create("task", map[string]any{"title": "Call the bank", "due": time.Date(y.Year(), y.Month(), y.Day(), 15, 0, 0, 0, time.Local).Format(time.RFC3339)})
	page := get(t, h, "/today").Body.String()
	tasks := page[strings.Index(page, ">Tasks<"):]
	if !strings.Contains(tasks, "Pay rent") || strings.Contains(tasks[:strings.Index(tasks, "</li>")], "<time") {
		t.Errorf("a whole day due today is today's, with no time: %s", truncate(tasks))
	}
	late := page[strings.Index(page, ">Late<"):strings.Index(page, ">Tasks<")]
	if !strings.Contains(late, `datetime="`+now.AddDate(0, 0, -3).Format("2006-01-02")+`"`) {
		t.Errorf("a late whole day says its day, its date as the value: %s", late)
	}
	if !strings.Contains(late, "Yesterday at ") {
		t.Errorf("a late task with a time says its day and time: %s", late)
	}
	if !strings.Contains(page, `<time datetime="`+now.Format("2006-01-02")+`">`+now.Format("Monday 2 January")+`</time>`) {
		t.Error("the page's day is a <time>")
	}
	if !strings.Contains(page, `href="/today" aria-current="page"`) {
		t.Error("Today is in the navigation, current on its own page")
	}
	// An agent reading the page through look hears each task with its day,
	// as a person reads it, not only its presses.
	if looked := get(t, h, "/api/look?only=text&path=/today").Body.String(); !strings.Contains(looked, "Call the bank Yesterday at ") {
		t.Errorf("look reads a task with its day: %s", truncate(looked))
	}

	// Moved to tomorrow, a whole day stays a whole day.
	tasksNow, _ := a.Store.List("task", store.ListOptions{})
	for _, r := range tasksNow {
		if r.Fields["title"] == "Renew passport" {
			postForm(t, h, "/today/move", url.Values{"id": {r.ID}, "to": {"tomorrow"}})
			got, _ := a.Store.Get("task", r.ID)
			due, _ := got.Fields["due"].(string)
			if d := now.AddDate(0, 0, 1).Format("2006-01-02"); due != d && due != d+"T00:00:00Z" {
				t.Errorf("moved, a whole day stays one: %q", due)
			}
		}
	}
}

// The brief interrupts, so a day with nothing on sends none, and a long
// day's brief names the three Today starts with.
func TestTheBriefSaysNothingOnAnEmptyDayAndNamesTheThree(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	h := server.New(a)
	var sent []string
	h.OnRing(func(title, text, link string) { sent = append(sent, title+" | "+text) })
	postForm(t, h, "/brief", url.Values{"at": {"00:00"}})
	now := time.Now()
	if h.BriefIfDue(now) {
		t.Error("nothing on, no brief")
	}
	a2, _ := newApp(t)
	h2 := server.New(a2)
	h2.OnRing(func(title, text, link string) { sent = append(sent, title+" | "+text) })
	postForm(t, h2, "/brief", url.Values{"at": {"00:00"}})
	for i, title := range []string{"Renew passport", "Pay rent", "Call the bank", "Book dentist", "Reply to the school"} {
		a2.Store.Create("task", map[string]any{"title": title, "due": now.AddDate(0, 0, -1-i).Format("2006-01-02")})
	}
	a2.Store.Create("task", map[string]any{"title": "Send the tax form", "due": now.Format("2006-01-02"), "tags": []any{"important"}})
	if page := get(t, h2, "/today").Body.String(); !strings.Contains(page, `3<span class="sw-visually-hidden"> tasks</span>`) {
		t.Errorf("the rest are counted as tasks, plural: %s", truncate(page))
	}
	if !h2.BriefIfDue(now) {
		t.Fatal("a long day sends its brief")
	}
	time.Sleep(50 * time.Millisecond)
	if len(sent) != 1 || !strings.Contains(sent[0], "Start with Send the tax form; Reply to the school; Book dentist") {
		t.Errorf("the brief names the three Today starts with: %v", sent)
	}
}
