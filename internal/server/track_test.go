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

// A habit is tracked out of the box: the provided habit and entry types,
// a tracker block that shows each against its target with its streak and
// a Log press, and the habit's own page with the numbers and a chart with
// the target drawn across it.
func TestAHabitIsTrackedOutOfTheBox(t *testing.T) {
	a, h := newApp(t)
	water, err := a.Store.Create(server.HabitType, map[string]any{"name": "Water", "cadence": "day", "target": 8, "unit": "glasses"})
	if err != nil {
		t.Fatal(err)
	}
	stretch, _ := a.Store.Create(server.HabitType, map[string]any{"name": "Stretch"})
	a.Store.Create(server.HabitType, map[string]any{"name": "Old", "archived": true})
	blk, err := a.Store.Create(chat.BlockType, a.Chat.BlockFields(map[string]any{"component": "tracker", "props": map[string]any{}}))
	if err != nil {
		t.Fatal(err)
	}
	// Yesterday and the day before were done; today is not yet.
	for days := 1; days <= 2; days++ {
		at := when.Store(time.Now().AddDate(0, 0, -days), false)
		a.Store.Create(server.EntryType, map[string]any{"habit": stretch.ID, "at": at})
		a.Store.Create(server.EntryType, map[string]any{"habit": water.ID, "at": at, "amount": 8})
	}
	a.Store.Create(server.EntryType, map[string]any{"habit": water.ID, "at": when.Store(time.Now(), false), "amount": 3})

	page := get(t, h, "/").Body.String()
	for _, want := range []string{`data-component="tracker"`, `data-id="` + water.ID + `" data-met="false"`, `3 of 8 glasses`, `2 days in a row`, `aria-valuenow="3"`, `aria-valuemax="8"`, `data-id="` + stretch.ID + `"`, `not yet`, `action="/habit/` + water.ID + `/log"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the tracker shows each habit against its target with its streak and a Log press, missing %q\n%s", want, page)
		}
	}
	if strings.Contains(page, ">Old<") {
		t.Error("an archived habit is left out")
	}

	// Log presses: five more glasses meets the target; a plain done for Stretch.
	post := func(id, amount string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/habit/"+id+"/log", strings.NewReader(url.Values{"amount": {amount}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Referer", "http://example.com/canvas/"+blk.ID)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if rec := post(water.ID, "5"); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/canvas/"+blk.ID {
		t.Errorf("logging goes back to the page it came from, got %d to %q", rec.Code, rec.Header().Get("Location"))
	}
	post(stretch.ID, "")
	entries, _ := a.Store.List(server.EntryType, store.ListOptions{})
	if len(entries) != 7 {
		t.Fatalf("each press is an entry record, got %d", len(entries))
	}
	page = get(t, h, "/").Body.String()
	for _, want := range []string{`data-id="` + water.ID + `" data-met="true"`, `8 of 8 glasses`, `data-id="` + stretch.ID + `" data-met="true"`, `3 days in a row`, `>done<`} {
		if !strings.Contains(page, want) {
			t.Errorf("after logging, both are met and the streak counts today, missing %q\n%s", want, page)
		}
	}
	if rec := post(water.ID, "lots"); rec.Code != http.StatusSeeOther {
		t.Errorf("a bad amount goes back with a notice, got %d", rec.Code)
	}

	// The habit's own page: its row, the best run, and a chart with the target.
	page = get(t, h, "/t/"+server.HabitType+"/"+water.ID).Body.String()
	for _, want := range []string{`id="habit-standing"`, `Best run: 3 days in a row.`, `data-component="chart"`, `class="sw-chart__target"`, `target 8 glasses`, `The last 30 days`} {
		if !strings.Contains(page, want) {
			t.Errorf("a habit's page has where it stands and a chart with the target drawn, missing %q\n%s", want, page)
		}
	}
}
