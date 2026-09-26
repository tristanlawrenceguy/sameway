package server_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// Tests for chips not repeating as dl fields on record detail pages.
// These pin that state and date shown in chips are not also repeated
// as separate <dt>/<dd> fields below, covering acceptance items 2 and 4.

// TestDetailChipsNotRepeatedInFields checks that a task with done=true
// and a due date does not repeat those facts in the dl on normal pages
// or even on ?show=fields pages — because chips + heading already said them.
func TestDetailChipsNotRepeatedInFields(t *testing.T) {
	a, h := newApp(t)

	// Create a task with done=true and a due date in the future.
	dueDate := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	rec, err := a.Store.Create("task", map[string]any{
		"title": "Water the plants",
		"done":  true,
		"due":   dueDate,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, view := range []string{"", fieldsView} {
		page := get(t, h, "/t/task/"+rec.ID+view).Body.String()

		// The done state should be present in the lede — as a badge chip when no
		// mark checkbox, or inside the mark checkbox label. Check for either pattern.
		hasDone := strings.Contains(page, `name="prop-done"`) || (strings.Contains(page, "sw-badge--success") && strings.Contains(page, ">Done"))
		if !hasDone {
			t.Errorf("normal page should have a Done chip\n%s", truncate(page))
		}

		// The Due date chip is rendered as a badge. Check for the sw-badge class and
		// the "Due" label text — the actual HTML uses multiple space-separated classes.
		hasDueChip := strings.Contains(page, `<span class="sw-when"`) || (strings.Contains(page, `class="sw-badge`) && strings.Contains(page, ">Due "))
		if !hasDueChip {
			t.Errorf("normal page should have a Due date chip\n%s", truncate(page))
		}

		// But the dl must not repeat "Done" or "Due" as field labels.
		body := extractBody(page)
		if strings.Contains(body, "<dt>Done</dt>") {
			t.Errorf("the dl should not have a <dt>Done</dt> — Done is already in chips\n%s", truncate(body))
		}
		if strings.Contains(body, "<dt>Due</dt>") {
			t.Errorf("the dl should not have a <dt>Due</dt> — Due is already in chips\n%s", truncate(body))
		}

		// The title must also not repeat in the dl.
		if strings.Contains(body, "<dt>Title</dt>") {
			t.Errorf("the dl should not have a <dt>Title</dt> — title is the h1\n%s", truncate(body))
		}
	}
}

// TestDetailNormalPageNoChipsRepeatedInFields checks that on a normal page
// (without ?show=fields), state and date chips are excluded from the dl.
func TestDetailNormalPageNoChipsRepeated(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title":  "Meeting notes",
		"status": "published",
		"pinned": true,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/note/"+rec.ID).Body.String()
	body := extractBody(page)

	// The dl must not repeat the enum (status/published) or bool (pinned/yes).
	if strings.Contains(body, "<dt>Status</dt>") {
		t.Errorf("the dl should not have <dt>Status</dt> — status is in chips\n%s", truncate(body))
	}
	if strings.Contains(body, "<dt>Pinned</dt>") {
		t.Errorf("the dl should not have <dt>Pinned</dt> — pinned is in chips\n%s", truncate(body))
	}

	// The title must not repeat either.
	if strings.Contains(body, "<dt>Title</dt>") {
		t.Errorf("the dl should not have <dt>Title</dt>\n%s", truncate(body))
	}

	// But the body (markdown) and tags should still be there if they exist.
	if !strings.Contains(page, "Meeting notes") {
		t.Errorf("the page should contain the title\n%s", truncate(page))
	}
}

// TestDetailFieldsViewNoChipsRepeated checks that even with ?show=fields,
// chips facts are not repeated in the dl. This is the key fix: previously
// ?show=fields would show all fields including title/bool/enum/datetime.
func TestDetailFieldsViewNoChipsRepeated(t *testing.T) {
	a, h := newApp(t)

	// Create a task with done=true and due date to hit all headFields types.
	dueDate := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	rec, err := a.Store.Create("task", map[string]any{
		"title": "Buy groceries",
		"done":  true,
		"due":   dueDate,
	})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/task/"+rec.ID+fieldsView).Body.String()
	body := extractBody(page)

	// All headFields must be excluded from the dl even with ?show=fields.
	for _, label := range []string{"<dt>Title</dt>", "<dt>Done</dt>", "<dt>Due</dt>"} {
		if strings.Contains(body, label) {
			t.Errorf("the fields view should not have %s — already in chips/heading\n%s", label, truncate(body))
		}
	}

	// The page should still be a valid HTML with the title.
	if !strings.Contains(page, "<h1") || !strings.Contains(page, "Buy groceries") {
		t.Errorf("the fields view page should have an h1 with the title\n%s", truncate(page))
	}
}

// TestAllRecordTypesNoTitleInCrumbs checks that every record type's detail
// page crumb does not repeat the title, covering acceptance item 4 for all types.
func TestAllRecordTypesNoTitleInCrumbs(t *testing.T) {
	a, h := newApp(t)

	// Create one record of each schema type that has a title field.
	var records []struct{ typ, id, title string }

	note, err := a.Store.Create("note", map[string]any{"title": "Alpha note"})
	if err == nil {
		records = append(records, struct{ typ, id, title string }{"note", note.ID, "Alpha note"})
	}

	task, err := a.Store.Create("task", map[string]any{"title": "Beta task"})
	if err == nil {
		records = append(records, struct{ typ, id, title string }{"task", task.ID, "Beta task"})
	}

	person, err := a.Store.Create("person", map[string]any{"name": "Charlie", "email": "charlie@example.com"})
	if err == nil {
		records = append(records, struct{ typ, id, title string }{"person", person.ID, "Charlie"})
	}

	action, err := a.Store.Create("action", map[string]any{
		"title": "Gamma action",
	})
	if err == nil {
		records = append(records, struct{ typ, id, title string }{"action", action.ID, "Gamma action"})
	}

	for _, r := range records {
		rec := get(t, h, "/t/"+r.typ+"/"+r.id)
		wantStatus(t, rec, http.StatusOK)
		body := extractBody(rec.Body.String())

		if strings.Contains(body, ">"+r.title+"</li>") ||
			strings.Contains(rec.Body.String(), `aria-current="page">`+r.title) {
			t.Errorf("/t/%s/%s: crumb should not repeat title %q\n%s", r.typ, r.id, r.title, truncate(rec.Body.String()))
		}

		// The h1 must still carry the title.
		if !strings.Contains(rec.Body.String(), ">"+r.title+"</h1>") {
			t.Errorf("/t/%s/%s: h1 should contain title %q\n%s", r.typ, r.id, r.title, truncate(rec.Body.String()))
		}

		// The crumb must still link to the listing.
		if !strings.Contains(rec.Body.String(), `href="/t/`+r.typ+`"`) {
			t.Errorf("/t/%s/%s: crumb should link back to /t/"+r.typ+"\n%s", r.typ, r.id, truncate(rec.Body.String()))
		}
	}

	// Also verify the listing link label is present (e.g., "Notes").
	noteRec := get(t, h, "/t/note/"+records[0].id)
	if !strings.Contains(noteRec.Body.String(), `href="/t/note">Notes</a>`) {
		t.Errorf("crumb should link to /t/note with label Notes\n%s", truncate(noteRec.Body.String()))
	}
}

// TestDetailPageHasCrumbs checks that every detail page still has a crumb
// navigation even after removing the title from it.
func TestDetailPageHasCrumbs(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{"title": "Epsilon note"})
	if err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/t/note/"+rec.ID).Body.String()

	if !strings.Contains(page, `aria-label="Breadcrumb"`) {
		t.Errorf("the detail page crumb should have an aria-label\n%s", truncate(page))
	}

	if !strings.Contains(page, `<nav class="sw-crumbs"`) {
		t.Errorf("the detail page should have a nav.sw-crumbs element\n%s", truncate(page))
	}

	// The listing link must be present.
	if !strings.Contains(page, `href="/t/note">Notes</a>`) {
		t.Errorf("crumb should contain the listing link to Notes\n%s", truncate(page))
	}
}
