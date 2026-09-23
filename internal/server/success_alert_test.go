package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestRecordPropsSaveShowsSuccessAlert checks that after POSTing to a note's
// props endpoint, the redirect target page contains a green .sw-alert with
// "Changes saved" text and a close button. Acceptance items 1 and 3.
func TestRecordPropsSaveShowsSuccessAlert(t *testing.T) {
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
	wantStatus(t, r, http.StatusSeeOther)

	follow := after(t, h, r)
	body := follow.Body.String()

	if !strings.Contains(body, "sw-alert") {
		t.Fatalf("saved detail page should contain an .sw-alert; body: %s", truncate(body))
	}
	if !strings.Contains(body, `data-kind="success"`) {
		t.Fatalf(`detail page after save should have kind="success"; body: %s`, truncate(body))
	}
	if !strings.Contains(body, "Changes saved") {
		t.Fatalf("saved detail page should contain \"Changes saved\"; body: %s", truncate(body))
	}
	if !strings.Contains(body, `data-dismiss`) {
		t.Fatalf("success alert should have a close button with data-dismiss; body: %s", truncate(body))
	}
	if !strings.Contains(body, "sw-alert__close") {
		t.Fatalf("success alert should have class sw-alert__close on the button; body: %s", truncate(body))
	}

	// Verify the update actually took effect.
	updated, err := a.Store.Get("note", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Fields["title"] != "Updated title" {
		t.Errorf("stored title = %q, want \"Updated title\"", updated.Fields["title"])
	}

	doc := parse(t, follow)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestRecordPropsSaveWithoutQueryParamShowsNoAlert checks that a detail page
// without ?saved does not show the success alert. This ensures ?saved is
// required for the alert to appear and it doesn't leak on unrelated requests.
func TestRecordPropsSaveWithoutQueryParamShowsNoAlert(t *testing.T) {
	a, h := newApp(t)

	rec, err := a.Store.Create("note", map[string]any{
		"title": "A note",
	})
	if err != nil {
		t.Fatal(err)
	}

	r := get(t, h, "/t/note/"+rec.ID)
	body := r.Body.String()

	if strings.Contains(body, `data-kind="success"`) && strings.Contains(body, "Changes saved") {
		t.Fatalf("detail page without ?saved must not show success alert; body: %s", truncate(body))
	}
}
