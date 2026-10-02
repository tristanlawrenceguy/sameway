package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Everything with a day is one calendar, whatever list it is in; a
// calendar and a collection on their own pages offer what they show, with
// the choices made on them.
func TestACalendarAndAListGoOutAsTheyAreShown(t *testing.T) {
	a, h := newApp(t)
<<<<<<< HEAD
	a.Store.Create("task", map[string]any{"title": "Dig the pond", "due": "2026-12-01T00:00:00Z"})
=======
	a.Store.Create("task", map[string]any{"title": "Dig the pond", "due": time.Now().AddDate(0, 0, 10).Format(time.RFC3339)})
>>>>>>> origin/main
	a.Store.Create("event", map[string]any{"title": "Harvest fair", "starts": "2026-10-12T00:00:00Z"})
	a.Store.Create("reminder", map[string]any{"title": "Water the beans", "at": "2026-10-02T07:00:00Z"})
	ics := get(t, h, "/export/all.ics").Body.String()
	for _, want := range []string{"SUMMARY:Dig the pond", "SUMMARY:Harvest fair", "SUMMARY:Water the beans"} {
		if strings.Count(ics, "BEGIN:VCALENDAR") != 1 || !strings.Contains(ics, want) {
			t.Errorf("one calendar with %q:\n%s", want, ics)
		}
	}

	cal := postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "calendar", "props": map[string]any{"type": "all"}})
	var blk struct{ ID string }
	decode(t, cal, &blk)
	if page := get(t, h, "/canvas/"+blk.ID).Body.String(); !strings.Contains(page, `href="/export/all.ics" download data-format="ics">Calendar (ICS)`) {
		t.Errorf("the whole calendar's page offers it as a calendar:\n%s", page)
	}

	seedTasks(t, a)
	id := addCollection(t, h, map[string]any{"type": "task", "where": []string{"done=false"}, "label": "Open", "controls": true})
	p := "c-" + id + "-"
	page := get(t, h, "/canvas/"+id+"?"+url.Values{p + "due": {"past"}}.Encode()).Body.String()
	want := `href="/export/task.csv?order=-created_at&amp;where=done%3Dfalse&amp;where=due%3Ctoday"`
	if !strings.Contains(page, want) || !strings.Contains(page, "Download these 2 tasks") {
		t.Errorf("the collection's page offers what it shows, narrowed as it is:\n%s", page)
	}
}
