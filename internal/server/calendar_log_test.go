package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A calendar of one habit's entries shows each as a link to its page, and
// its day view logs that habit for the day shown, back on the same day.
func TestACalendarOfEntriesLogsForTheDayShown(t *testing.T) {
	a, h := newApp(t)
	hours, _ := a.Store.Create(server.HabitType, map[string]any{"name": "Hours", "cadence": "month", "aim": "limit", "target": 21, "unit": "hours"})
	water, _ := a.Store.Create(server.HabitType, map[string]any{"name": "Water", "target": 8, "unit": "glasses"})
	day := time.Now().AddDate(0, 0, -3)
	on := day.Format("2006-01-02")
	entry, _ := a.Store.Create(server.EntryType, map[string]any{"habit": hours.ID, "at": when.Store(day, true), "amount": 2})
	cal, err := a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "calendar", "props": map[string]any{"type": "entry", "where": []any{"habit=" + hours.ID}}}))
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/canvas/"+cal.ID+"?day="+on).Body.String()
	for _, want := range []string{`href="/t/entry/` + entry.ID + `"`, `class="sw-calendar__add"`, `aria-label="Log for ` + day.Format("Mon 2 Jan") + `"`, `action="/habit/` + hours.ID + `/log"`, `name="on" type="text" placeholder="today" value="` + on + `"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the day view links the entry and logs its habit for that day, missing %q", want)
		}
	}
	if strings.Contains(page, `action="/habit/`+water.ID+`/log"`) {
		t.Error("a calendar of one habit offers that habit alone")
	}

	req := httptest.NewRequest(http.MethodPost, "/habit/"+hours.ID+"/log", strings.NewReader(url.Values{"amount": {"3"}, "on": {on}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "http://example.com/canvas/"+cal.ID+"?day="+on)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Location") != "/canvas/"+cal.ID+"?day="+on {
		t.Errorf("logging goes back to the day, got %q", rec.Header().Get("Location"))
	}
	entries, _ := a.Store.List(server.EntryType, store.ListOptions{})
	logged := false
	for _, e := range entries {
		if e.Fields["amount"] == 3.0 && e.Fields["at"] == on+"T00:00:00Z" {
			logged = true
		}
	}
	if !logged {
		t.Errorf("the entry is on the day shown, got %+v", entries)
	}

	// A calendar of words, not records, offers nothing to log.
	words, _ := a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "calendar", "props": map[string]any{"events": []any{map[string]any{"date": on, "label": "Dentist"}}}}))
	if page := get(t, h, "/canvas/"+words.ID+"?day="+on).Body.String(); strings.Contains(page, "sw-calendar__add") {
		t.Error("a calendar without a type offers nothing to log")
	}
}
