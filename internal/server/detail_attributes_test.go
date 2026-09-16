package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

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

// TestDetailPageDataPropOnEditableFields checks that each editable field's
// <dd> element carries data-prop matching the field name.
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
	doc := parse(t, get(t, h, "/t/note/"+rec.ID))

	for _, prop := range []string{"title", "body", "tags", "status", "pinned"} {
		dd := doc.WithAttr("data-prop", prop)
		if len(dd) == 0 {
			t.Errorf("<dd data-prop=%q should exist for note detail page\n%s", prop, truncate(get(t, h, "/t/note/"+rec.ID).Body.String()))
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

// TestDetailPageHasEditButton checks that the detail page renders an Edit
// button with data-inline-edit alongside the Delete link.
func TestDetailPageHasEditButton(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/note/"+rec.ID))

	buttons := doc.WithAttr("data-inline-edit", "")
	if len(buttons) == 0 {
		t.Errorf("detail page should have a button with data-inline-edit\n%s", truncate(get(t, h, "/t/note/"+rec.ID).Body.String()))
	}
}

// TestDetailPageEditButtonText checks that the Edit button text is "Edit note"
// for note detail pages.
func TestDetailPageEditButtonText(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/note/"+rec.ID))

	buttons := doc.WithAttr("data-inline-edit", "")
	if len(buttons) == 0 {
		t.Fatalf("no data-inline-edit button found for text check")
	}
	text := strings.TrimSpace(htmltest.Text(buttons[0]))
	want := "Edit note"
	if text != want {
		t.Errorf("Edit button text = %q, want %q\n%s", text, want, truncate(get(t, h, "/t/note/"+rec.ID).Body.String()))
	}
}

// TestDetailPageEditButtonIsPlainHTML checks that the Edit button is a plain
// HTML <button type="button"> element — no JavaScript dependency for its existence.
func TestDetailPageEditButtonIsPlainHTML(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	wantButton := `<button type="button" data-inline-edit>Edit note</button>`
	if !strings.Contains(body, wantButton) {
		t.Errorf("body should contain exact button markup %q\n%s", wantButton, truncate(body))
	}
}

// TestDetailPageEditButtonInCluster checks that the Edit button is in the same
// sw-cluster div as the Delete link.
func TestDetailPageEditButtonInCluster(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Test note"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID).Body.String()

	if !strings.Contains(body, `class="sw-cluster"`) {
		t.Error("detail page should have a sw-cluster div\n" + truncate(body))
	}

	editIdx := strings.Index(body, `<button type="button" data-inline-edit>Edit note</button>`)
	deleteIdx := strings.Index(body, "Delete note")
	if editIdx < 0 {
		t.Errorf("Edit button not found in body\n%s", truncate(body))
	}
	if deleteIdx < 0 {
		t.Errorf("Delete link text not found in body\n%s", truncate(body))
	}

	clusterOpen := strings.Index(body, `class="sw-cluster"`)
	if clusterOpen >= 0 && editIdx > 0 {
		closeCluster := strings.Index(body[clusterOpen:], "</div>")
		if closeCluster < 0 || editIdx > clusterOpen+closeCluster {
			t.Errorf("Edit button should be inside the sw-cluster div\n%s", truncate(body))
		}
	}
}
