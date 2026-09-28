package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// bodyText is what a person reads on a page: the body's text, without
// what is only for a screen reader.
func bodyText(t *testing.T, doc *htmltest.Doc) string {
	t.Helper()
	bodies := doc.Elements("body")
	if len(bodies) == 0 {
		t.Fatal("no body")
	}
	return htmltest.VisibleText(bodies[0])
}

// TestTypeNamesReadInWords: a type is stored as test_type and said as
// "test type" wherever a person reads it: the add button, the heading,
// the count, the new record's title, its Delete button, its crumb and its
// line in the activity log (backlog 0544).
func TestTypeNamesReadInWords(t *testing.T) {
	a, h := newApp(t)
	postJSON(t, h, http.MethodPost, "/api/types", map[string]any{
		"name": "test_type", "description": "A test type.", "title": "title",
		"fields": []map[string]any{{"name": "title", "type": "string"}, {"name": "priority", "type": "int"}},
	})
	if _, ok := a.Types.Get("test_type"); !ok {
		t.Fatal("test_type was not made")
	}

	list := bodyText(t, parse(t, get(t, h, "/t/test_type")))
	for _, want := range []string{"Add test type", "Test types"} {
		if !strings.Contains(list, want) {
			t.Errorf("the list should say %q\n%s", want, truncate(list))
		}
	}

	added := postForm(t, h, "/t/test_type/add", url.Values{})
	where := added.Header().Get("Location")
	if where == "" {
		t.Fatalf("adding gave %d and no page to go to", added.Code)
	}
	detail := bodyText(t, parse(t, get(t, h, strings.SplitN(where, "?", 2)[0])))
	for _, want := range []string{"New test type", "Delete test type"} {
		if !strings.Contains(detail, want) {
			t.Errorf("the new record's page should say %q\n%s", want, truncate(detail))
		}
	}
	activity := bodyText(t, parse(t, get(t, h, "/activity")))
	for page, text := range map[string]string{"list": list, "record": detail, "activity": activity} {
		if strings.Contains(text, "test_type") {
			t.Errorf("the %s page shows the stored name test_type\n%s", page, text)
		}
	}
}

// TestEmptyRecordShowsNoFieldKinds: a record with nothing but its title
// shows its title, not a list of its type's field kinds as if they were
// its values; there is no special page for a type called test_type.
func TestEmptyRecordShowsNoFieldKinds(t *testing.T) {
	a, h := newApp(t)
	postJSON(t, h, http.MethodPost, "/api/types", map[string]any{
		"name": "test_type", "description": "A test type.", "title": "title",
		"fields": []map[string]any{{"name": "title", "type": "string"}, {"name": "priority", "type": "int"}, {"name": "body_text", "type": "text"}},
	})
	empty, err := a.Store.Create("test_type", map[string]any{"title": "My Test"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/test_type/"+empty.ID+fieldsView).Body.String()
	for _, kind := range []string{">int</dd>", ">text</dd>", ">string</dd>"} {
		if strings.Contains(body, kind) {
			t.Errorf("an empty record should not list its fields' kinds, as %s\n%s", kind, truncate(body))
		}
	}
	full, err := a.Store.Create("test_type", map[string]any{"title": "Full", "body_text": "some text"})
	if err != nil {
		t.Fatal(err)
	}
	body = get(t, h, "/t/test_type/"+full.ID+fieldsView).Body.String()
	if !strings.Contains(body, "<dt>Body text</dt>") || !strings.Contains(body, "some text") {
		t.Errorf("a record's values should be listed\n%s", truncate(body))
	}
}

// TestDetailEditNamedForRecord: the Edit button on a record's page is
// named for the record, "Edit Buy milk", not "Edit block": the block the
// inline editor arms carries the title, as a canvas block does (backlog
// 0550). The name is heard, not seen, so the page still says the title once.
func TestDetailEditNamedForRecord(t *testing.T) {
	a, h := newApp(t)
	rec, err := a.Store.Create("note", map[string]any{"title": "Buy milk", "body": "Two litres."})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/note/"+rec.ID))
	blocks := doc.WithAttr("data-block-id", rec.ID)
	if len(blocks) != 1 {
		t.Fatalf("want the record's one block, got %d", len(blocks))
	}
	if got, _ := htmltest.Attr(blocks[0], "data-block-label"); got != "Buy milk" {
		t.Errorf("data-block-label = %q, want the record's title", got)
	}
	if n := strings.Count(bodyText(t, doc), "Buy milk"); n != 1 {
		t.Errorf("the page should show the title once, shows it %d times", n)
	}
}

// TestActivityPageSaysNoAddress: /activity does not show /api/activity
// as words; an agent finds the log from the page's alternate link
// (backlog 0555).
func TestActivityPageSaysNoAddress(t *testing.T) {
	_, h := newApp(t)
	doc := parse(t, get(t, h, "/activity"))
	if text := bodyText(t, doc); strings.Contains(text, "/api/") {
		t.Errorf("the activity page shows an address\n%s", text)
	}
	found := false
	for _, l := range doc.Elements("link") {
		if rel, _ := htmltest.Attr(l, "rel"); rel == "alternate" {
			if href, _ := htmltest.Attr(l, "href"); href == "/api/activity" {
				found = true
			}
		}
	}
	if !found {
		t.Error(`the activity page should keep <link rel="alternate" href="/api/activity">`)
	}
}

// TestHabitPageSaysItsSettingsInWords: a habit's page names its settings
// in words, How much and Counted in, not target and unit made capital,
// and a goal of 0 is no goal, so it says nothing of one (backlog 0551).
func TestHabitPageSaysItsSettingsInWords(t *testing.T) {
	a, h := newApp(t)
	rec, err := a.Store.Create("habit", map[string]any{"name": "Water", "target": 8.0, "unit": "glasses", "goal": 0.0})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/habit/"+rec.ID+fieldsView).Body.String()
	for _, want := range []string{"<dt>How much</dt>", "<dt>Counted in</dt>", "<dt>Aiming for</dt>", "<dt>Entries in a period</dt>"} {
		if !strings.Contains(body, want) {
			t.Errorf("the habit's page should say %s", want)
		}
	}
	for _, not := range []string{"<dt>Target</dt>", "<dt>Aim</dt>", "<dt>Combine</dt>", "<dt>Goal</dt>", "<dt>Longer goal</dt>"} {
		if strings.Contains(body, not) {
			t.Errorf("the habit's page should not say %s", not)
		}
	}
	if _, err := a.Store.Update("habit", rec.ID, map[string]any{"goal": 100.0}); err != nil {
		t.Fatal(err)
	}
	body = get(t, h, "/t/habit/"+rec.ID+fieldsView).Body.String()
	if !strings.Contains(body, "<dt>Longer goal</dt>") {
		t.Error("a habit with a goal should say it")
	}
}
