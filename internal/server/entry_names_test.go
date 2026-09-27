package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// An entry has no title of its own: it is named by its habit and how
// much in the log, in its list and on its page, never by its id; its day
// is when it happened, not a day it "was" due; and a long habit name is
// cut rather than the amount.
func TestAnEntryIsNamedByItsHabit(t *testing.T) {
	a, h := newApp(t)
	water, err := a.Store.Create(server.HabitType, map[string]any{"name": "8 glasses of water a day", "unit": "glasses", "target": 8})
	if err != nil {
		t.Fatal(err)
	}
	at := when.Store(time.Now().Add(-time.Minute), false)
	wantStatus(t, postJSON(t, h, "POST", "/api/entry", map[string]any{"habit": water.ID, "at": at, "amount": 1}), 201)

	log := get(t, h, "/activity").Body.String()
	if !strings.Contains(log, "8 glasses of water a day: 1 glass") {
		t.Error("the log names an entry by its habit and how much")
	}
	list := get(t, h, "/t/"+server.EntryType).Body.String()
	if !strings.Contains(list, "8 glasses of water…: 1 glass") {
		t.Error("a row cuts a long habit name and keeps how much")
	}
	if strings.Contains(list, "Was at") {
		t.Error("an entry is done: its day is when it happened, not overdue")
	}
}
