package server_test

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// Today puts what is on today on one page, late first; the morning brief
// sends the same at the time set, once a day, leading to Today.
func TestTodayAndItsMorningBrief(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	now := time.Now()
	today, yesterday, tomorrow := now.Format("2006-01-02"), now.AddDate(0, 0, -1).Format("2006-01-02"), now.AddDate(0, 0, 1).Format("2006-01-02")
	for _, r := range []struct {
		typ    string
		fields map[string]any
	}{
		{"task", map[string]any{"title": "Pay rent", "due": today}},
		{"task", map[string]any{"title": "Renew passport", "due": yesterday}},
		{"task", map[string]any{"title": "Book holiday", "due": tomorrow}},
		{"task", map[string]any{"title": "Done already", "due": today, "done": true}},
		{"event", map[string]any{"title": "Dentist", "starts": today + " 23:30"}},
	} {
		if _, err := a.Store.Create(r.typ, r.fields); err != nil {
			t.Fatal(err)
		}
	}
	h := server.New(a)
	var sent []string
	h.OnRing(func(title, text, link string) { sent = append(sent, title+" | "+text+" | "+link) })
	page := get(t, h, "/today").Body.String()
	for _, want := range []string{"Pay rent", "Renew passport", "Dentist", ">Late<"} {
		if !strings.Contains(page, want) {
			t.Errorf("Today has %s: %s", want, truncate(page))
		}
	}
	if strings.Contains(page, "Book holiday") || strings.Contains(page, "Done already") {
		t.Error("and not tomorrow's, nor what is done")
	}
	postForm(t, h, "/brief", url.Values{"at": {"00:00"}})
	if a.Workspace.Config.Brief.At != "00:00" {
		t.Fatalf("the brief's time is kept: %q", a.Workspace.Config.Brief.At)
	}
	if !h.BriefIfDue(now) || h.BriefIfDue(now) {
		t.Fatal("sent once a day")
	}
	time.Sleep(50 * time.Millisecond)
	if len(sent) != 1 || !strings.Contains(sent[0], "Today: 1 task, 1 event, 1 late") || !strings.HasSuffix(sent[0], "/today") {
		t.Errorf("the brief says what is on and leads to Today: %v", sent)
	}
}
