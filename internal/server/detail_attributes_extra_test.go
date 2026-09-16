package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestDetailPageActivityDataAttributes checks that activity detail pages also
// get data-block-id, data-edit-action, and data-prop attributes.
func TestDetailPageActivityDataAttributes(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("activity", map[string]any{
		"actor":  "human",
		"action": "added",
		"detail": "A test activity",
	})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/activity/"+rec.ID))

	blocks := doc.WithAttr("data-block-id", rec.ID)
	if len(blocks) == 0 {
		t.Errorf("activity detail should have a div with data-block-id=%q\n%s", rec.ID, truncate(get(t, h, "/t/activity/"+rec.ID).Body.String()))
	}

	if len(blocks) == 0 {
		t.Fatalf("no wrapping div found for data-edit-action check")
	}
	action, ok := htmltest.Attr(blocks[0], "data-edit-action")
	if !ok {
		t.Errorf("activity wrapping div should have data-edit-action\n%s", truncate(get(t, h, "/t/activity/"+rec.ID).Body.String()))
		return
	}
	want := "/t/activity/" + rec.ID + "/props"
	if action != want {
		t.Errorf("data-edit-action = %q, want %q", action, want)
	}

	for _, prop := range []string{"actor", "action", "detail"} {
		dd := doc.WithAttr("data-prop", prop)
		if len(dd) == 0 {
			t.Errorf("<dd data-prop=%q should exist for activity detail\n%s", prop, truncate(get(t, h, "/t/activity/"+rec.ID).Body.String()))
		}
	}

	buttons := doc.WithAttr("data-inline-edit", "")
	if len(buttons) == 0 {
		t.Errorf("activity detail should have a data-inline-edit button")
	} else {
		text := strings.TrimSpace(htmltest.Text(buttons[0]))
		if text != "Edit activity" {
			t.Errorf("Edit button text = %q, want \"Edit activity\"", text)
		}
	}

	for _, prop := range []string{"Created", "Updated"} {
		dd := doc.WithAttr("data-prop", prop)
		if len(dd) > 0 {
			t.Errorf("<dd should not have data-prop=%q on activity\n%s", prop, truncate(get(t, h, "/t/activity/"+rec.ID).Body.String()))
		}
	}
}

// TestDetailPageProposalDataAttributes checks that proposal detail pages also
// get the same data attributes pattern.
func TestDetailPageProposalDataAttributes(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("proposal", map[string]any{
		"summary": "Should we refactor this?",
		"action":  map[string]any{"tool": "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/proposal/"+rec.ID))

	blocks := doc.WithAttr("data-block-id", rec.ID)
	if len(blocks) == 0 {
		t.Errorf("proposal detail should have a div with data-block-id=%q\n%s", rec.ID, truncate(get(t, h, "/t/proposal/"+rec.ID).Body.String()))
	}

	if len(blocks) == 0 {
		t.Fatalf("no wrapping div found for data-edit-action check")
	}
	action, ok := htmltest.Attr(blocks[0], "data-edit-action")
	if !ok {
		t.Errorf("proposal wrapping div should have data-edit-action\n%s", truncate(get(t, h, "/t/proposal/"+rec.ID).Body.String()))
		return
	}
	want := "/t/proposal/" + rec.ID + "/props"
	if action != want {
		t.Errorf("data-edit-action = %q, want %q", action, want)
	}

	for _, prop := range []string{"summary", "action"} {
		dd := doc.WithAttr("data-prop", prop)
		if len(dd) == 0 {
			t.Errorf("<dd data-prop=%q should exist for proposal detail\n%s", prop, truncate(get(t, h, "/t/proposal/"+rec.ID).Body.String()))
		}
	}

	buttons := doc.WithAttr("data-inline-edit", "")
	if len(buttons) == 0 {
		t.Errorf("proposal detail should have a data-inline-edit button")
	} else {
		text := strings.TrimSpace(htmltest.Text(buttons[0]))
		if text != "Edit proposal" {
			t.Errorf("Edit button text = %q, want \"Edit proposal\"", text)
		}
	}

	for _, prop := range []string{"Created", "Updated"} {
		dd := doc.WithAttr("data-prop", prop)
		if len(dd) > 0 {
			t.Errorf("<dd should not have data-prop=%q on proposal\n%s", prop, truncate(get(t, h, "/t/proposal/"+rec.ID).Body.String()))
		}
	}
}

// TestDetailPageStatusFieldHasDataProp checks that the status enum field on a
// note gets data-prop="status" even though its default value is "draft".
func TestDetailPageStatusFieldHasDataProp(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Draft note"})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/note/"+rec.ID))

	dd := doc.WithAttr("data-prop", "status")
	if len(dd) == 0 {
		t.Errorf("<dd data-prop=\"status\" should exist for note with default status\n%s", truncate(get(t, h, "/t/note/"+rec.ID).Body.String()))
	}
}

// TestDetailPageDataBlockIdUniqueness checks that each record has its own
// unique data-block-id — no cross-contamination between records.
func TestDetailPageDataBlockIdUniqueness(t *testing.T) {
	a, h := newApp(t)

	rec1, err := a.Store.Create("note", map[string]any{"title": "First"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.Store.Create("note", map[string]any{"title": "Second"}); err != nil {
		t.Fatal(err)
	}

	doc1 := parse(t, get(t, h, "/t/note/"+rec1.ID))
	blocks1 := doc1.WithAttr("data-block-id", "")
	if len(blocks1) == 0 {
		t.Fatalf("no data-block-id found to test uniqueness")
	}
	if len(blocks1) != 1 {
		t.Errorf("exactly one data-block-id per page, got %d", len(blocks1))
	}
	if id, _ := htmltest.Attr(blocks1[0], "data-block-id"); id != rec1.ID {
		t.Errorf("data-block-id = %q, want %q", id, rec1.ID)
	}

	doc2 := parse(t, get(t, h, "/t/note/"+rec1.ID))
	blocks2 := doc2.WithAttr("data-block-id", "")
	if len(blocks2) == 0 {
		t.Fatalf("no data-block-id found on second load")
	}
	if len(blocks2) != 1 {
		t.Errorf("exactly one data-block-id per page, got %d", len(blocks2))
	}
	if id, _ := htmltest.Attr(blocks2[0], "data-block-id"); id != rec1.ID {
		t.Errorf("page for rec1 should show rec1's ID as data-block-id")
	}
}

// TestDetailPageBodyFieldHasDataProp checks that the body field on a note gets
// data-prop="body". The body is markdown but still editable.
func TestDetailPageBodyFieldHasDataProp(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Note with body",
		"body":  "# Hello\n\nSome markdown.",
	})
	if err != nil {
		t.Fatal(err)
	}
	doc := parse(t, get(t, h, "/t/note/"+rec.ID))

	dd := doc.WithAttr("data-prop", "body")
	if len(dd) == 0 {
		t.Errorf("<dd data-prop=\"body\" should exist for note with body content\n%s", truncate(get(t, h, "/t/note/"+rec.ID).Body.String()))
	}
}
