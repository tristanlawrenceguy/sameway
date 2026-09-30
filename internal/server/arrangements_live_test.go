package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
)

// Words the arrangements once put on pages as examples; the evaluation's
// models left them there (runs A-t1, B-t1, B-t6).
var placeholders = []string{"go here", "The next book", "The one after", "Your first book", "Start here", "An idea to come back to", "Book the dentist", "Walking shoes", "What it is and where you are in it"}

func arrange(t *testing.T, a *app.App, name string) string {
	t.Helper()
	said, isErr := call(t, a, "add_arrangement", map[string]any{"name": name})
	if isErr {
		t.Fatalf("add_arrangement %s: %s", name, said)
	}
	return said
}

func noPlaceholders(t *testing.T, page string) {
	t.Helper()
	for _, p := range placeholders {
		if strings.Contains(page, p) {
			t.Errorf("the page holds the example words %q", p)
		}
	}
}

// On an empty workspace every arrangement's blocks say there is nothing
// yet, in the empty component's words, and the result says what each
// block shows: nothing is invented to fill the page.
func TestArrangementsOnAnEmptyWorkspaceSayThereIsNothingYet(t *testing.T) {
	a, h := newApp(t)
	for name, shows := range map[string][]string{
		"week": {"calendar: added calendar", "it shows 0 events by starts", "todo: added collection", "it shows 0 tasks, not done", "habits: added tracker", "it shows 0 habits"},
		"desk": {"drafts: added collection", "it shows 0 notes", "pinned: added collection"},
		"trip": {"dates: added calendar", "it shows 0 events", "pack: added collection", "it shows 0 tasks", "book: added collection"},
	} {
		said := arrange(t, a, name)
		for _, s := range shows {
			if !strings.Contains(said, s) {
				t.Errorf("%s: the result should say %q, got %q", name, s, said)
			}
		}
	}
	page := get(t, h, "/").Body.String()
	noPlaceholders(t, page)
	for _, empty := range []string{"Nothing coming up", "Nothing matches: ", "Nothing tracked yet"} {
		if !strings.Contains(page, empty) {
			t.Errorf("an empty block should say %q", empty)
		}
	}
}

// On a workspace with records, the same arrangements show them.
func TestArrangementsShowThePersonsRecords(t *testing.T) {
	a, h := newApp(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	for _, r := range []struct {
		typ    string
		fields map[string]any
	}{
		{"task", map[string]any{"title": "Call the plumber", "due": tomorrow}},
		{"task", map[string]any{"title": "Sun cream", "tags": []any{"packing"}}},
		{"task", map[string]any{"title": "Sleeper train", "tags": []any{"booking"}, "due": tomorrow}},
		{"event", map[string]any{"title": "Dentist at nine", "starts": tomorrow}},
		{"note", map[string]any{"title": "Chapter one", "body": "It was a bright cold day.", "status": "draft"}},
		{"note", map[string]any{"title": "Names for characters", "pinned": true, "status": "published"}},
		{"habit", map[string]any{"name": "Drink water", "unit": "glasses"}},
	} {
		if _, err := a.Store.Create(r.typ, r.fields); err != nil {
			t.Fatal(err)
		}
	}
	week := arrange(t, a, "week")
	for _, s := range []string{"it shows 1 event by starts", "it shows 2 tasks, not done", "it shows 1 habit"} {
		if !strings.Contains(week, s) {
			t.Errorf("week: the result should say %q, got %q", s, week)
		}
	}
	desk := arrange(t, a, "desk")
	trip := arrange(t, a, "trip")
	if !strings.Contains(desk, "it shows 1 note") || !strings.Contains(trip, "it shows 1 task") {
		t.Errorf("each block says how many of the person's records it shows: %q %q", desk, trip)
	}
	page := get(t, h, "/").Body.String()
	noPlaceholders(t, page)
	for _, want := range []string{"Call the plumber", "Sun cream", "Sleeper train", "Dentist at nine", "Chapter one", "It was a bright cold day.", "Names for characters", "Drink water"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page should show the person's %q", want)
		}
	}
}

// Reading shows book records. With no book type it adds nothing and says
// how to make one; with a book type missing status it says which field;
// made as it says, the arrangement goes on.
func TestReadingSaysWhatTheWorkspaceLacks(t *testing.T) {
	a, h := newApp(t)
	before := blockCount(t, a)
	said, isErr := call(t, a, "add_arrangement", map[string]any{"name": "reading"})
	if !isErr || !strings.Contains(said, "nothing added") || !strings.Contains(said, "no book type") || !strings.Contains(said, `add_type {"description"`) || !strings.Contains(said, `"name":"book"`) {
		t.Errorf("reading with no book type should say how to make one, got %q", said)
	}
	if n := blockCount(t, a); n != before {
		t.Errorf("nothing is added: %d blocks, were %d", n, before)
	}

	if said, isErr := call(t, a, "add_type", map[string]any{"name": "book", "properties": []any{map[string]any{"name": "title", "kind": "string"}}}); isErr {
		t.Fatal(said)
	}
	said, isErr = call(t, a, "add_arrangement", map[string]any{"name": "reading"})
	if !isErr || !strings.Contains(said, "book has no status field") || !strings.Contains(said, `add_field {`) {
		t.Errorf("reading with a book type lacking status should say to add it, got %q", said)
	}
	if said, isErr := call(t, a, "add_field", map[string]any{"type": "book", "name": "status", "kind": "enum", "values": []any{"to_read", "reading", "read"}, "default": "to_read"}); isErr {
		t.Fatal(said)
	}
	said = arrange(t, a, "reading")
	if !strings.Contains(said, "list: added collection") || !strings.Contains(said, "it shows 0 books") {
		t.Errorf("reading should say what each block shows, got %q", said)
	}
	page := get(t, h, "/").Body.String()
	noPlaceholders(t, page)
	if !strings.Contains(page, "Nothing matches: ") {
		t.Error("with no books, reading says nothing matches")
	}
	if _, err := a.Store.Create("book", map[string]any{"title": "Middlemarch", "status": "reading"}); err != nil {
		t.Fatal(err)
	}
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, "Middlemarch") {
		t.Error("a book being read shows under Now reading")
	}
}
