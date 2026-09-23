package server_test

import (
	"net/url"
	"strings"
	"testing"
)

// TestRecordPropsSaveNoSuccessPrefix checks that the success alert shown after
// saving a note's props does NOT contain the "Success:" prefix. The green
// styling, role="status", and text "Changes saved" communicate success without
// a redundant label. Acceptance item 1.
func TestRecordPropsSaveNoSuccessPrefix(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "Original title",
	})
	if err != nil {
		t.Fatal(err)
	}

	form := url.Values{}
	form.Set("prop-title", "Updated title")
	r := postForm(t, h, "/t/note/"+rec.ID+"/props", form)
	wantStatus(t, r, 303) // See Other redirect

	follow := after(t, h, r)
	body := follow.Body.String()

	// The "Success:" prefix must NOT appear in the success alert.
	if strings.Contains(body, `>Success:</`) {
		t.Fatalf("success alert must not contain \"Success:\" label; body:\n%s", truncate(body))
	}

	// The remaining success signals must still be present.
	if !strings.Contains(body, "Changes saved") {
		t.Fatalf("alert should still say \"Changes saved\"; body:\n%s", truncate(body))
	}
	if !strings.Contains(body, `data-kind="success"`) {
		t.Fatalf(`alert missing data-kind="success"; body: %s`, truncate(body))
	}
}

// TestCanvasEditNoSuccessPrefix checks that the success alert shown after a
// canvas inline-edit does NOT contain the "Success:" prefix. Acceptance item 1.
func TestCanvasEditNoSuccessPrefix(t *testing.T) {
	h, id := canvasWithABlock(t)

	rec := postForm(t, h, "/canvas/"+id+"/props", url.Values{"prop-title": {"Groceries"}})

	page := after(t, h, rec)
	body := page.Body.String()

	if strings.Contains(body, `>Success:</`) {
		t.Fatalf("success alert must not contain \"Success:\" label; body:\n%s", truncate(body))
	}

	if !strings.Contains(body, "Changes saved") {
		t.Fatalf("alert should still say \"Changes saved\"; body:\n%s", truncate(body))
	}
}
