package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A record's connections.
//
// The system works out everything a record joins on to, in both
// directions and past the refs: what points here, what is set about this
// page, what sits beside it under the same parent, what else falls on its
// day. The page says how many of each and nothing else. The address opens
// one where the person already is. The API hands over the whole graph
// with the query that follows each one, because the assistant is the one
// deciding whether the person has a reason to see any of it.
func TestConnectionsAreCountedOnThePageAndWholeInTheAPI(t *testing.T) {
	_, h := newApp(t)
	var garden, pond, garlic, liner struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden"}), &garden)
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Dig the pond", "project": garden.ID, "due": "2026-10-01T00:00:00Z"}), &pond)
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Plant garlic", "project": garden.ID}), &garlic)
	decode(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{"title": "Buy a liner", "at": "2026-10-01T12:00:00Z", "about": "/t/task/" + pond.ID}), &liner)

	page := get(t, h, "/t/task/"+pond.ID).Body.String()
	rel := page[strings.Index(page, ">Related</h2>"):]
	for _, want := range []string{
		">1 other task in Garden</a>",
		">1 reminder about this</a>",
		">1 reminder on Thu 1 Oct 2026</a>",
	} {
		if !strings.Contains(rel, want) {
			t.Errorf("the page counts this connection, missing %q\n%s", want, rel)
		}
	}
	// Counted, not listed. The records themselves are not on the page for
	// anybody: what a sighted person cannot see is not in the accessibility
	// tree either, and the link that opens it is the same link for both.
	for _, gone := range []string{"Plant garlic", "Buy a liner"} {
		if strings.Contains(rel, gone) {
			t.Errorf("%q is not on the page until someone asks for it\n%s", gone, rel)
		}
	}

	// Asking opens that one connection where the person already is, with
	// the way back out, and the others stay a line of counts.
	open := get(t, h, "/t/task/"+pond.ID+"?show=alongside:task.project").Body.String()
	shown := open[strings.Index(open, `data-related="alongside:task.project"`):]
	if !strings.Contains(shown, "Plant garlic") || strings.Contains(shown, "Dig the pond") {
		t.Errorf("the opened list is the others in the project, not this one\n%s", shown)
	}
	if !strings.Contains(shown, ">Hide<") || !strings.Contains(shown, `href="/t/task/`+pond.ID+`"`) {
		t.Errorf("an opened connection can be closed again\n%s", shown)
	}
	if !strings.Contains(open, ">1 reminder about this</a>") {
		t.Error("opening one connection leaves the others counted")
	}

	// Two at once, because the address holds what is open: the assistant
	// hands over a page with the connections it has a reason to show.
	both := get(t, h, "/t/task/"+pond.ID+"?show=alongside:task.project&show=about:reminder.about").Body.String()
	if !strings.Contains(both, "Plant garlic") || !strings.Contains(both, "Buy a liner") {
		t.Errorf("an address can open more than one connection\n%s", both[strings.Index(both, ">Related</h2>"):])
	}

	// The API gets all of it: the kind, why, the count, the ids, and the
	// query that lists them.
	var rec struct {
		Related []struct {
			Key, Kind, Type, Why string
			Where                []string
			Count                int
			IDs                  []string
		}
	}
	decode(t, get(t, h, "/api/task/"+pond.ID), &rec)
	if len(rec.Related) != 3 {
		t.Fatalf("every connection is in the API, got %+v", rec.Related)
	}
	found := false
	for _, l := range rec.Related {
		if l.Key != "alongside:task.project" {
			continue
		}
		found = true
		if l.Count != 1 || len(l.IDs) != 1 || l.IDs[0] != garlic.ID || l.Why != "their project is the same project" {
			t.Errorf("the connection says what it is and what is on it: %+v", l)
		}
		// The same query, run by the list page, gives the same records:
		// the count a person sees and the list it opens are one query, not
		// a filter applied somewhere they cannot see it.
		list := get(t, h, "/t/task?where="+strings.Join(l.Where, "&where=")).Body.String()
		if !strings.Contains(list, "Plant garlic") || strings.Contains(list, "Dig the pond") {
			t.Errorf("the connection's query lists the same records: %v\n%s", l.Where, list)
		}
	}
	if !found {
		t.Errorf("the records beside this one under the same parent are a connection: %+v", rec.Related)
	}
}

// A record with nothing joined to it says nothing about connections.
func TestARecordWithNoConnectionsSaysNothing(t *testing.T) {
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "On its own"}), &note)
	if page := get(t, h, "/t/note/"+note.ID).Body.String(); strings.Contains(page, ">Related</h2>") {
		t.Errorf("no connections, no section\n%s", page)
	}
}
