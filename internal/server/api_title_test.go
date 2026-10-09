package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A record over the API says its title, the one its page is headed with
// (crew findings 0562 and 0564): an entry is called by its habit and how
// much, a note by its title, in the list, the single record, and what a
// write answers with; the fields are where they always were.
func TestAPIRecordSaysItsTitle(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	habit, err := a.Store.Create(server.HabitType, map[string]any{"name": "Read", "unit": "minutes", "cadence": "day"})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := a.Store.Create(server.EntryType, map[string]any{"habit": habit.ID, "at": when.Store(time.Now().Add(-time.Minute), false), "amount": 25.0})
	if err != nil {
		t.Fatal(err)
	}

	var one struct {
		Title  string         `json:"title"`
		Fields map[string]any `json:"fields"`
	}
	json.Unmarshal(get(t, h, "/api/entry/"+entry.ID).Body.Bytes(), &one)
	if one.Title != "Read: 25 minutes" || one.Fields["habit"] != habit.ID {
		t.Errorf("an entry says what its page is headed with, its fields unchanged: %+v", one)
	}

	type listed struct {
		Records []struct {
			ID     string         `json:"id"`
			Title  string         `json:"title"`
			Fields map[string]any `json:"fields"`
		} `json:"records"`
	}
	var entries listed
	json.Unmarshal(get(t, h, "/api/entry").Body.Bytes(), &entries)
	if len(entries.Records) != 1 || entries.Records[0].Title != "Read: 25 minutes" || entries.Records[0].Fields == nil {
		t.Errorf("a listed entry says its title: %+v", entries)
	}

	made := postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "Call the dentist", "body": "Ask about Thursday."})
	var note struct {
		ID     string         `json:"id"`
		Title  string         `json:"title"`
		Fields map[string]any `json:"fields"`
	}
	json.Unmarshal(made.Body.Bytes(), &note)
	if made.Code != http.StatusCreated || note.Title != "Call the dentist" || note.Fields["body"] != "Ask about Thursday." {
		t.Errorf("a create answers with the record and its title: %d %s", made.Code, made.Body)
	}

	changed := postJSON(t, h, http.MethodPatch, "/api/note/"+note.ID, map[string]any{"title": "Call the dentist again"})
	json.Unmarshal(changed.Body.Bytes(), &note)
	if changed.Code != http.StatusOK || note.Title != "Call the dentist again" {
		t.Errorf("an update answers with the new title: %d %s", changed.Code, changed.Body)
	}
	json.Unmarshal(get(t, h, "/api/note/"+note.ID).Body.Bytes(), &one)
	if one.Title != "Call the dentist again" {
		t.Errorf("a note says its title: %+v", one)
	}
	var notes listed
	json.Unmarshal(get(t, h, "/api/note").Body.Bytes(), &notes)
	if len(notes.Records) != 1 || notes.Records[0].Title != "Call the dentist again" {
		t.Errorf("a listed note says its title: %+v", notes)
	}

	moved := postJSON(t, h, http.MethodPatch, "/api/entry/"+entry.ID, map[string]any{"amount": 40.0})
	json.Unmarshal(moved.Body.Bytes(), &one)
	if moved.Code != http.StatusOK || one.Title != "Read: 40 minutes" {
		t.Errorf("an entry's update answers with the title worked out again: %d %s", moved.Code, moved.Body)
	}
}
