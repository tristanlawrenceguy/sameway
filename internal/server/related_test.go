package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A record's connections, and the parts of its page that are off.
//
// The system works out everything a record joins on to: what points here,
// what is set about this page, what sits beside it under the same parent,
// what else falls on its day. None of it is on the page. Not the records,
// not a count of them, not a link — a page at rest is the record and
// nothing about what else exists. The address opens one, the workspace
// keeps one on, and the API hands the whole graph to the assistant, which
// is the thing that knows to offer.
func TestAPageAtRestSaysNothingAboutWhatElseExists(t *testing.T) {
	_, h := newApp(t)
	var garden, pond, garlic, liner struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden"}), &garden)
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Dig the pond", "project": garden.ID, "due": "2026-10-01T00:00:00Z"}), &pond)
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Plant garlic", "project": garden.ID}), &garlic)
	decode(t, postJSON(t, h, http.MethodPost, "/api/reminder", map[string]any{"title": "Buy a liner", "at": "2026-10-01T12:00:00Z", "about": "/t/task/" + pond.ID}), &liner)

	page := get(t, h, "/t/task/"+pond.ID).Body.String()
	for _, gone := range []string{"Plant garlic", "Buy a liner", ">Related</h2>", "other task in Garden", "?show=", `aria-label="Do with this"`} {
		if strings.Contains(page, gone) {
			t.Errorf("a page at rest says nothing about what else exists, found %q\n%s", gone, page)
		}
	}
	// It says the record once. The heading, the chips and the words are
	// the page; the list below them used to say the same things again.
	if strings.Contains(page, "<dt>Title</dt>") || strings.Contains(page, "<dt>Due</dt>") || strings.Contains(page, "<dt>Done</dt>") {
		t.Errorf("the field list leaves out what the heading and chips already say\n%s", page)
	}

	// Asking opens one connection where the person already is, with the
	// way back out, and opens nothing else.
	open := get(t, h, "/t/task/"+pond.ID+"?show=alongside:task.project").Body.String()
	shown := open[strings.Index(open, `data-related="alongside:task.project"`):]
	if !strings.Contains(shown, "Plant garlic") || strings.Contains(shown, "Dig the pond") {
		t.Errorf("the opened list is the others in the project, not this one\n%s", shown)
	}
	if !strings.Contains(shown, ">Fewer<") || !strings.Contains(shown, `href="/t/task/`+pond.ID+`"`) {
		t.Errorf("what the address opened, a link closes\n%s", shown)
	}
	if strings.Contains(open, "Buy a liner") {
		t.Error("opening one connection opens only that one")
	}

	// Two at once, because the address holds what is open: the assistant
	// hands over a page with exactly what it has a reason to show.
	both := get(t, h, "/t/task/"+pond.ID+"?show=alongside:task.project&show=about:reminder.about").Body.String()
	if !strings.Contains(both, "Plant garlic") || !strings.Contains(both, "Buy a liner") {
		t.Errorf("an address can open more than one connection\n%s", both)
	}
	// And the whole record minus what the heading and chips already said.
	fields := get(t, h, "/t/task/"+pond.ID+"?show=fields").Body.String()
	if !strings.Contains(fields, "<dt>Project</dt>") || !strings.Contains(fields, ">Fewer<") {
		t.Errorf("show=fields shows all non-chip fields and closes again\n%s", fields)
	}
	if strings.Contains(fields, "<dt>Title</dt>") || strings.Contains(fields, "<dt>Done</dt>") || strings.Contains(fields, "<dt>Due</dt>") {
		t.Errorf("show=fields still leaves out what chips already said\n%s", fields)
	}
}

// The page says none of it; the API says all of it. That is the trade:
// the assistant is the way in, so it is given the whole graph, with the
// query that follows each connection and the count on the other end.
func TestTheAPICarriesEveryConnectionWhole(t *testing.T) {
	a, h := newApp(t)
	garden, _ := a.Store.Create("project", map[string]any{"title": "Garden"})
	pond, _ := a.Store.Create("task", map[string]any{"title": "Dig the pond", "project": garden.ID})
	garlic, _ := a.Store.Create("task", map[string]any{"title": "Plant garlic", "project": garden.ID})

	var rec struct {
		Related []struct {
			Key, Kind, Type, Why string
			Where                []string
			Count                int
			IDs                  []string
		}
	}
	decode(t, get(t, h, "/api/task/"+pond.ID), &rec)
	found := false
	for _, l := range rec.Related {
		if l.Key != "alongside:task.project" {
			continue
		}
		found = true
		if l.Count != 1 || len(l.IDs) != 1 || l.IDs[0] != garlic.ID || l.Why != "their project is the same project" {
			t.Errorf("a connection says what it is and what is on it: %+v", l)
		}
		// The same query, run by the list page, gives the same records:
		// what the API promises is what following it produces.
		list := get(t, h, "/t/task?where="+strings.Join(l.Where, "&where=")).Body.String()
		if !strings.Contains(list, "Plant garlic") || strings.Contains(list, "Dig the pond") {
			t.Errorf("the connection's query lists the same records: %v\n%s", l.Where, list)
		}
	}
	if !found {
		t.Errorf("the records beside this one under the same parent are a connection: %+v", rec.Related)
	}
}

// What the assistant turns on stays on, and carries no control for
// turning it off: it goes the way it came, by asking.
func TestTheAssistantTurnsAPartOnForGood(t *testing.T) {
	a, h := newApp(t)
	var garden, pond struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/project", map[string]any{"title": "Garden"}), &garden)
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Dig the pond", "project": garden.ID}), &pond)

	if err := a.Workspace.Set("ui.show", "+points-here:task.project"); err != nil {
		t.Fatalf("turning a part on: %v", err)
	}
	if err := a.Workspace.Set("ui.show", "+remind"); err != nil {
		t.Fatalf("a second part is added, not swapped in: %v", err)
	}
	if got := a.Workspace.Config.UI.Show; got != "points-here:task.project, remind" {
		t.Errorf("ui.show keeps what is already there, got %q", got)
	}
	page := get(t, h, "/t/project/"+garden.ID).Body.String()
	if !strings.Contains(page, "Dig the pond") {
		t.Errorf("a part the workspace turned on is on without asking\n%s", page)
	}
	if strings.Contains(page, ">Fewer<") {
		t.Error("it carries no way off: that is asking, not a control on every page")
	}
	if task := get(t, h, "/t/task/"+pond.ID).Body.String(); !strings.Contains(task, `name="about" value="/t/task/`+pond.ID+`"`) {
		t.Errorf("remind is on every record's page once it is turned on\n%s", task)
	}

	if err := a.Workspace.Set("ui.show", "-remind"); err != nil {
		t.Fatalf("taking one back: %v", err)
	}
	if got := a.Workspace.Config.UI.Show; got != "points-here:task.project" {
		t.Errorf("-key takes back one and leaves the rest, got %q", got)
	}
	if task := get(t, h, "/t/task/"+pond.ID).Body.String(); strings.Contains(task, `aria-label="Do with this"`) {
		t.Errorf("and the page is calm again\n%s", task)
	}
}

// A record with nothing joined to it and nothing asked for is just itself.
func TestARecordWithNoConnectionsSaysNothing(t *testing.T) {
	_, h := newApp(t)
	var note struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/note", map[string]any{"title": "On its own", "body": "Some words."}), &note)
	page := get(t, h, "/t/note/"+note.ID).Body.String()
	for _, gone := range []string{">Related</h2>", "<dl class=\"sw-dl\">", `aria-label="Do with this"`} {
		if strings.Contains(page, gone) {
			t.Errorf("nothing to say, nothing said, found %q\n%s", gone, page)
		}
	}
	if !strings.Contains(page, "Some words.") {
		t.Errorf("what the person came for is still there\n%s", page)
	}
}
