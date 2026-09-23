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

// A habit can be a monthly limit or a measure only recorded: the tracker
// says what is left or over, a record has no bar, and the habit's page
// logs for an earlier day and draws the limit on a monthly chart.
func TestALimitAndARecordAreTracked(t *testing.T) {
	a, h := newApp(t)
	hours, err := a.Store.Create(server.HabitType, map[string]any{"name": "Hours", "cadence": "month", "aim": "limit", "target": 21, "unit": "hours"})
	if err != nil {
		t.Fatal(err)
	}
	weight, err := a.Store.Create(server.HabitType, map[string]any{"name": "Weight", "aim": "record", "combine": "latest", "unit": "kg"})
	if err != nil {
		t.Fatal(err)
	}
	a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "tracker", "props": map[string]any{}}))
	now := time.Now()
	a.Store.Create(server.EntryType, map[string]any{"habit": hours.ID, "at": when.Store(now, true), "amount": 15})
	a.Store.Create(server.EntryType, map[string]any{"habit": weight.ID, "at": when.Store(now.Add(-time.Minute), false), "amount": 73.1})
	a.Store.Create(server.EntryType, map[string]any{"habit": weight.ID, "at": when.Store(now, false), "amount": 72.8})

	page := get(t, h, "/").Body.String()
	for _, want := range []string{`data-id="` + hours.ID + `" data-met="true" data-aim="limit"`, `15 of 21 hours, 6 left`, `Hours this month`, `data-aim="record"`, `72.8 kg`} {
		if !strings.Contains(page, want) {
			t.Errorf("a limit says what is left and a record its latest reading, missing %q", want)
		}
	}
	if strings.Contains(page, `aria-label="Weight this day`) {
		t.Error("a record has no target, so no bar")
	}

	// Logging from the habit's page for an earlier day this month.
	page = get(t, h, "/t/"+server.HabitType+"/"+hours.ID).Body.String()
	for _, want := range []string{`name="on" type="date"`, `The last 12 months`, `limit 21 hours`} {
		if !strings.Contains(page, want) {
			t.Errorf("the habit's page offers a day to log for and a monthly chart with the limit, missing %q", want)
		}
	}
	day := now.AddDate(0, 0, -1)
	if day.Month() != now.Month() {
		day = now
	}
	req := httptest.NewRequest(http.MethodPost, "/habit/"+hours.ID+"/log", strings.NewReader(url.Values{"amount": {"8"}, "on": {day.Format("2006-01-02")}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(httptest.NewRecorder(), req)
	entries, _ := a.Store.List(server.EntryType, store.ListOptions{})
	found := false
	for _, e := range entries {
		at, _ := e.Fields["at"].(string)
		if e.Fields["amount"] == 8.0 && (day.Equal(now) || at == day.Format("2006-01-02")+"T00:00:00Z") {
			found = true
		}
	}
	if !found {
		t.Errorf("the entry is on the day given, got %+v", entries)
	}
	page = get(t, h, "/").Body.String()
	if !strings.Contains(page, `23 of 21 hours, 2 over`) || !strings.Contains(page, `data-id="`+hours.ID+`" data-met="false" data-aim="limit"`) {
		t.Errorf("past the limit the tracker says how far over\n%s", page)
	}
}
