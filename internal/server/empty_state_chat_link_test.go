package server_test

// Tests for empty-state links pointing to /chat instead of dead /t/{type}/new routes.
// These fail today (the code still uses /t/note/new) and pass once the developer
// changes views.go line 57 to link to /chat with text "Ask the assistant…".

import (
	"net/http"
	"strings"
	"testing"
)

// TestEmptyStateLinksToChat checks that an empty notes listing page links to
// /chat (the working surface for creating content), not the dead /t/note/new.
// Covers Acceptance 1 and 2 of task 0395.
func TestEmptyStateLinksToChat(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// The empty-state paragraph must link to /chat (possibly with query params).
	if !strings.Contains(body, `<a href="/chat`) {
		t.Errorf("empty-state should link to /chat (the working surface for creation)\n%s", truncate(body))
	}

	// It must NOT link to the dead /t/note/new route.
	if strings.Contains(body, `/t/note/new`) {
		t.Error("empty-state must not link to /t/note/new — that route returns 404")
	}
}

// TestEmptyStateSaysAskTheAssistant checks that the empty-state text says
// "Ask the assistant" rather than just "Add your first", matching the product's
// design language of directing people to the chat. Covers Acceptance 3.
func TestEmptyStateSaysAskTheAssistant(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// Two ways that work: by hand, which needs no model, and by asking.
	if !strings.Contains(body, "ask the assistant</a>") || !strings.Contains(body, `action="/t/note/add"`) {
		t.Errorf("the empty list offers adding one by hand and asking the assistant\n%s", truncate(body))
	}

	// The old phrasing must be gone.
	if strings.Contains(body, `>Add your first <a href="/t/`) {
		t.Error("empty-state must not use the old 'Add your first ... /t/{type}/new' pattern")
	}
}

// TestEmptyStateLinksToChatForActions checks that the actions listing also
// links to /chat (not a dead form route). Covers Acceptance 1 for the action type.
func TestEmptyStateLinksToChatForActions(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/action")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `<a href="/chat`) {
		t.Errorf("empty-state for actions should link to /chat\n%s", truncate(body))
	}

	if strings.Contains(body, `/t/action/new`) {
		t.Error("empty-state must not link to /t/action/new — that route returns 404")
	}
}

// TestEmptyStateLinksToChatForTasks checks that the tasks listing also links
// to /chat (not a dead form route). Covers Acceptance 1 for the task type.
func TestEmptyStateLinksToChatForTasks(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/task")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `<a href="/chat`) {
		t.Errorf("empty-state for tasks should link to /chat\n%s", truncate(body))
	}

	if strings.Contains(body, `/t/task/new`) {
		t.Error("empty-state must not link to /t/task/new — that route returns 404")
	}
}
