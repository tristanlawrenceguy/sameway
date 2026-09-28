package server_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// fieldsView asks a record's page for its whole field list. A page at
// rest leaves out what its heading and chips already say, so the marks
// the inline editor works from live on this view: see parts.go.
const fieldsView = "?show=fields"

// TestDetailPageHasDataBlockID checks that a note detail page wraps its
// definition list in a div with data-block-id set to the record ID.
func TestDetailPageHasDataBlockID(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/note/"+rec.ID))

	blocks := doc.WithAttr("data-block-id", rec.ID)
	if len(blocks) == 0 {
		t.Errorf("detail page should have a div with data-block-id=%q\n%s", rec.ID, truncate(get(t, h, "/t/note/"+rec.ID).Body.String()))
	}
}

// TestDetailPageHasDataEditAction checks that the wrapping div on a note
// detail page carries data-edit-action pointing to the inline edit endpoint.
func TestDetailPageHasDataEditAction(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/note/"+rec.ID))

	blocks := doc.WithAttr("data-block-id", rec.ID)
	if len(blocks) == 0 {
		t.Fatalf("no wrapping div found for data-edit-action check")
	}
	action, ok := htmltest.Attr(blocks[0], "data-edit-action")
	if !ok {
		t.Errorf("wrapping div should have a data-edit-action attribute\n%s", truncate(get(t, h, "/t/note/"+rec.ID).Body.String()))
		return
	}
	want := "/t/note/" + rec.ID + "/props"
	if action != want {
		t.Errorf("data-edit-action = %q, want %q", action, want)
	}
}

// TestDetailPageDataPropOnEditableFields checks that editable fields' <dd>
// elements carry data-prop matching the field name. Title, bools, and enums
// are excluded from the dl (they appear as chips/heading) so only body and
// tags should have data-prop attributes on this view.
func TestDetailPageDataPropOnEditableFields(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title":  "My note",
		"body":   "Some body text",
		"tags":   []any{"a", "b"},
		"status": "draft",
		"pinned": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/note/"+rec.ID+fieldsView))

	for _, prop := range []string{"body", "tags"} {
		dd := doc.WithAttr("data-prop", prop)
		if len(dd) == 0 {
			t.Errorf("<dd data-prop=%q should exist for note detail page\n%s", prop, truncate(get(t, h, "/t/note/"+rec.ID+fieldsView).Body.String()))
		}
	}

	// Title (heading), status (enum chip), pinned (bool mark) are excluded.
	for _, prop := range []string{"title", "status", "pinned"} {
		dd := doc.WithAttr("data-prop", prop)
		if len(dd) > 0 {
			t.Errorf("<dd should not have data-prop=%q — it's in chips/heading\n%s", prop, truncate(get(t, h, "/t/note/"+rec.ID+fieldsView).Body.String()))
		}
	}
}

// TestDetailPageNoDataPropOnTimestamps checks that Created and Updated do NOT
// have data-prop attributes — they are not editable.
func TestDetailPageNoDataPropOnTimestamps(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/note/"+rec.ID))

	for _, prop := range []string{"Created", "Updated"} {
		dd := doc.WithAttr("data-prop", prop)
		if len(dd) > 0 {
			t.Errorf("<dd should not have data-prop=%q — timestamps are not editable\n%s", prop, truncate(get(t, h, "/t/note/"+rec.ID).Body.String()))
		}
	}
}
