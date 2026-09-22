package server_test

import (
	"net/url"
	"strings"
	"testing"
)

// TestCanvasEditShowsSuccessAlert checks that after a person inline-edits a
// canvas block, the reloaded home page shows a green .sw-alert with "Changes
// saved" text and a close button. Acceptance items 1 and 3.
func TestCanvasEditShowsSuccessAlert(t *testing.T) {
	h, id := canvasWithABlock(t)

	rec := postForm(t, h, "/canvas/"+id+"/props", url.Values{"prop-title": {"Groceries"}})

	page := get(t, h, rec.Header().Get("Location"))
	body := page.Body.String()

	if !strings.Contains(body, "sw-alert") {
		t.Fatalf("home page after inline edit should contain an .sw-alert; body: %s", truncate(body))
	}
	if !strings.Contains(body, `data-kind="success"`) {
		t.Fatalf(`home page after save should have kind="success"; body: %s`, truncate(body))
	}
	if !strings.Contains(body, "Changes saved") {
		t.Fatalf("home page after save should contain \"Changes saved\"; body: %s", truncate(body))
	}
	if !strings.Contains(body, `data-dismiss`) {
		t.Fatalf("success alert should have a close button with data-dismiss; body: %s", truncate(body))
	}
	if !strings.Contains(body, "sw-alert__close") {
		t.Fatalf("success alert should have class sw-alert__close on the button; body: %s", truncate(body))
	}

	doc := parse(t, page)
	assertAllComponentsKnown(t, doc, componentNames)
}

// TestCanvasEditWithoutSavedParamShowsNoAlert checks that a home page without
// ?saved does not show the success alert. This ensures ?saved is required for
// the alert to appear and it doesn't leak on unrelated requests.
func TestCanvasEditWithoutSavedParamShowsNoAlert(t *testing.T) {
	_, h := newApp(t)

	page := get(t, h, "/")
	body := page.Body.String()

	if strings.Contains(body, `data-kind="success"`) && strings.Contains(body, "Changes saved") {
		t.Fatalf("home page without ?saved must not show success alert; body: %s", truncate(body))
	}
}
