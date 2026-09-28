package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// TestTestTypeDetailShowsSchemaFieldsWhenEmpty verifies that a test_type
// record with only its title set renders the schema's field definitions in
// place of the empty definition list. This is the fallback for backlog 0464:
// when no record values exist beyond the title, the page should still show
// what fields are available.
func TestTestTypeDetailShowsSchemaFieldsWhenEmpty(t *testing.T) {
	a, h := newApp(t)

	// Create a test_type with two custom fields via API.
	postJSON(t, h, http.MethodPost, "/api/types", map[string]any{
		"name":        "test_type",
		"description": "A test type for testing.",
		"title":       "title",
		"fields": []map[string]any{
			{"name": "title", "type": "string"},
			{"name": "test_field", "type": "text"},
			{"name": "priority", "type": "int"},
		},
	})

	// Create a record with only the title set — no custom field values.
	rec, err := a.Store.Create("test_type", map[string]any{
		"title": "My Test",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/test_type/"+rec.ID+fieldsView).Body.String()

	// The schema field definitions should be shown as a definition list.
	if !strings.Contains(body, "<dt>Test field</dt>") {
		t.Errorf("detail page should show <dt>Test field</dt> from the schema\n%s", truncate(body))
	}
	if !strings.Contains(body, "<dt>Priority</dt>") {
		t.Errorf("detail page should show <dt>Priority</dt> from the schema\n%s", truncate(body))
	}

	// Title must not be repeated in the dl (it is the h1).
	if strings.Contains(body, "<dt>Title</dt>") {
		t.Error("detail page should not show <dt>Title</dt> — title is the heading")
	}
}

// TestTestTypeDetailShowsRecordValuesWhenSet verifies that when a test_type
// record has actual field values set, those values render in the definition
// list instead of the schema field definitions. The fallback only activates
// when there are no record values to display.
func TestTestTypeDetailShowsRecordValuesWhenSet(t *testing.T) {
	a, h := newApp(t)

	// Create a test_type with fields via API.
	postJSON(t, h, http.MethodPost, "/api/types", map[string]any{
		"name":        "test_type",
		"description": "A test type for testing.",
		"title":       "title",
		"fields": []map[string]any{
			{"name": "title", "type": "string"},
			{"name": "body_text", "type": "text"},
		},
	})

	// Create a record with custom field values.
	rec, err := a.Store.Create("test_type", map[string]any{
		"title":     "My Test",
		"body_text": "some text",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := get(t, h, "/t/test_type/"+rec.ID+fieldsView).Body.String()

	// The record's actual values should render in the definition list.
	if !strings.Contains(body, "<dt>Body text</dt>") || !strings.Contains(body, "some text") {
		t.Errorf("detail page should show the body_text value\n%s", truncate(body))
	}
}

// TestTestTypeDetailDoesNotAffectOtherTypes verifies that the schema field
// fallback for test_type does not change how other content types render.
// A note with only title set still shows no definition list — it is not
// affected by this change.
func TestTestTypeDetailDoesNotAffectOtherTypes(t *testing.T) {
	a, h := newApp(t)

	// Create a record with only title — body, tags etc are all nil/zero.
	rec, err := a.Store.Create("note", map[string]any{"title": "Just a title"})
	if err != nil {
		t.Fatal(err)
	}
	body := get(t, h, "/t/note/"+rec.ID+fieldsView).Body.String()

	// The note detail page should still skip empty fields and show no dl.
	if strings.Contains(body, "<dt>Tags</dt>") {
		t.Error("note detail page should not show <dt>Tags</dt> for an empty field")
	}
	// And it must NOT have schema definitions — notes have a pre-written schema.
	if !strings.Contains(body, "Created ") {
		t.Error("detail page should still say when the record was made")
	}
}
