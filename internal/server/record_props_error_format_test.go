package server_test

import (
	"net/url"
	"strings"
	"testing"
)

// TestRecordPropsValidationErrorFormatIsPlainSentence checks that validation
// errors in the props handler render as plain sentences ("Title is required")
// rather than the old colon-separated technical format ("<strong>title</strong>:
// is required"). This covers acceptance item 1.
func TestRecordPropsValidationErrorFormatIsPlainSentence(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Test note",
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("prop-title", "")
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	body := after(t, h, r).Body.String()

	// The new format should be a plain sentence: "Title is required".
	if !strings.Contains(body, "Title is required") {
		t.Errorf("error response should show 'Title is required' as a plain sentence,\n"+
			"but body does not contain it.\n"+
			"Body starts with: %q", truncate(body))
	}

	// The old format used <strong>title</strong>: which should no longer appear.
	if strings.Contains(body, "<strong>") && strings.Contains(body, "</strong>:") {
		t.Errorf("error response still uses the old '<strong>field</strong>: msg' format,\n"+
			"but it should render as a plain sentence like 'Title is required'.\n"+
			"Body starts with: %q", truncate(body))
	}
}

// TestRecordPropsValidationErrorUnknownFieldFormatIsPlainSentence checks that
// unknown field errors also use the plain-sentence format ("Bogus field unknown
// field") rather than "<strong>bogus_field</strong>: ...". This covers
// acceptance item 2.
func TestRecordPropsValidationErrorUnknownFieldFormatIsPlainSentence(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Test note",
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("prop-bogus_field", "nope")
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	body := after(t, h, r).Body.String()

	// The new format should be a plain sentence with spaces: "Bogus field".
	if !strings.Contains(body, "Bogus field") {
		t.Errorf("error response should show 'Bogus field' as a plain sentence,\n"+
			"but body does not contain it.\n"+
			"Body starts with: %q", truncate(body))
	}

	// The old format used underscores and <strong> tags.
	if strings.Contains(body, "<strong>bogus_field</strong>:") {
		t.Errorf("error response still uses the old '<strong>bogus_field</strong>: msg' format,\n"+
			"but it should render as 'Bogus field unknown field'.\n"+
			"Body starts with: %q", truncate(body))
	}
}

// TestRecordPropsValidationErrorEnumFormatIsPlainSentence checks that enum
// validation errors also use the plain-sentence format ("Status must be one of")
// rather than "<strong>status</strong>: ...". This covers acceptance item 2.
func TestRecordPropsValidationErrorEnumFormatIsPlainSentence(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Test note",
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("prop-status", "not_a_valid_status")
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	body := after(t, h, r).Body.String()

	// The new format should be a plain sentence: "Status must be one of".
	if !strings.Contains(body, "Status") {
		t.Errorf("error response should show 'Status' (capitalized),\n"+
			"but body does not contain it.\n"+
			"Body starts with: %q", truncate(body))
	}

	// The old format used <strong>status</strong>: which should no longer appear.
	if strings.Contains(body, "<strong>") && strings.Contains(body, "</strong>:") {
		t.Errorf("error response still uses the old '<strong>field</strong>: msg' format,\n"+
			"but it should render as a plain sentence.\n"+
			"Body starts with: %q", truncate(body))
	}
}
