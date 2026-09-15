package server_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// TestConfirmDeleteAlertKindIsWarning verifies that the delete confirmation
// page renders its alert with kind="warning" (not "danger"), so assistive
// technologies and screen readers announce a warning rather than an error.
// This covers acceptance item 1: internal/server/delete_confirm.go must pass
// "kind": "warning" when rendering the alert on the delete confirmation page.
func TestConfirmDeleteAlertKindIsWarning(t *testing.T) {
	a, h := newApp(t)
	noteID := createNote(t, h, a)

	path := "/t/note/" + noteID + "/confirm-delete"
	rec := get(t, h, path)
	wantStatus(t, rec, http.StatusOK)

	body := rec.Body.String()

	// The alert data-kind attribute must be "warning", not "danger".
	if !strings.Contains(body, `data-kind="warning"`) {
		t.Errorf("alert should have data-kind=\"warning\"; body contains %q", findAttrSnippet(body, "data-kind"))
	}
	if strings.Contains(body, `data-kind="danger"`) && strings.Contains(body, "/confirm-delete") {
		t.Errorf("delete confirmation alert must not use danger kind; found data-kind=\"danger\" on confirm-delete page")
	}

	// The rendered text should say "Warning:" not "Error:".
	if !strings.Contains(body, "Warning:") {
		t.Errorf("alert body should contain \"Warning:\", got: %s", truncate(body))
	}
	if strings.Contains(body, "Error:") {
		t.Errorf("delete confirmation alert must not show \"Error:\"; found in: %s", truncate(body))
	}

	// The full visible title text should be "Warning: Delete note?". Use
	// htmltest.Text to strip HTML tags so the span-wrapped kind label and
	// the title text appear as a single contiguous string.
	doc := parse(t, rec)
	bodyText := htmltest.Text(doc.Root)
	if !strings.Contains(bodyText, "Warning: Delete note?") {
		t.Errorf("alert title should read \"Warning: Delete note?\", got %q", truncate(bodyText))
	}
}

// TestConfirmDeleteAlertKindIsWarningForMessage verifies that message
// confirm-delete pages also use the warning kind. This ensures the change
// applies to all content types, not just notes.
func TestConfirmDeleteAlertKindIsWarningForMessage(t *testing.T) {
	a, h := newApp(t)

	// Create a message record.
	rec, err := a.Store.Create("message", map[string]any{
		"role":    "user",
		"content": "Test message for delete warning kind",
	})
	if err != nil {
		t.Fatal(err)
	}

	path := "/t/message/" + rec.ID + "/confirm-delete"
	resp := get(t, h, path)
	wantStatus(t, resp, http.StatusOK)

	body := resp.Body.String()

	if !strings.Contains(body, `data-kind="warning"`) {
		t.Errorf("message delete alert should have data-kind=\"warning\", got: %s", truncate(body))
	}
	if strings.Contains(body, "Error:") {
		t.Errorf("message delete alert must not show \"Error:\", got: %s", truncate(body))
	}

	// Use htmltest.Text so the span-wrapped kind label and title appear as
	// a single contiguous string of visible text.
	doc := parse(t, resp)
	bodyText := htmltest.Text(doc.Root)
	if !strings.Contains(bodyText, "Warning: Delete message?") {
		t.Errorf("alert title should read \"Warning: Delete message?\", got %q", truncate(bodyText))
	}
}

// findAttrSnippet returns a short snippet around the given attribute name for
// use in error messages. It is not part of the test logic itself.
func findAttrSnippet(body, attr string) string {
	idx := strings.Index(body, attr)
	if idx == -1 {
		return "(not found)"
	}
	start := idx
	if start > 80 {
		start = idx - 80
	}
	end := idx + 80
	if end > len(body) {
		end = len(body)
	}
	return body[start:end]
}
