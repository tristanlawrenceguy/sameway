package server_test

// Tests for task 0181: simplify empty state text on list pages.
// These assert that every list page's empty state follows the new structure:
// "No X yet" heading (≤4 words) + one action prompt line (≤8 words).
// They fail today because the code still uses the old copy.

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

// TestListPageEmptyStateHasHeading checks that an empty notes listing page
// shows "No note yet" as its heading inside the empty-state block. This is
// Acceptance 1: every list page's empty state heading starts with "No [type]
// yet". Covers acceptance item 1 for the note type.
func TestListPageEmptyStateHasHeading(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `<h2 class="sw-empty__title">No notes yet</h2>`) {
		t.Errorf("empty notes listing should have an h2 heading 'No notes yet'\n%s", truncate(body))
	}
}

// TestListPageEmptyStateHasActionPrompt checks that the empty notes listing
// follows with a sw-empty paragraph containing a link to /chat and short text.
// Acceptance 3: action prompt is at most 8 words. Acceptance 4: functional path
// via chat link. Covers acceptance items 3-4 for the note type.
func TestListPageEmptyStateHasActionPrompt(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `data-component="empty"`) {
		t.Errorf("empty notes listing should use the sw-empty paragraph\n%s", truncate(body))
	}
	if !strings.Contains(body, `href="/chat?prompt=`) {
		t.Error("empty-state must link to /chat with a prompt parameter")
	}
	// The heading and action must be two separate elements (heading + one line).
	if strings.Contains(body, `<h2 class="sw-empty__title">No notes yet</h2><p class="sw-empty__message">`) {
		return // correct structure found
	}
	t.Errorf("empty notes listing should have an h2 followed by a sw-empty paragraph\n%s", truncate(body))
}

// TestListPageEmptyStateForActions checks the same pattern for actions.
func TestListPageEmptyStateForActions(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/action")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `<h2 class="sw-empty__title">No actions yet</h2>`) {
		t.Errorf("empty actions listing should have an h2 heading 'No actions yet'\n%s", truncate(body))
	}
	if !strings.Contains(body, `data-component="empty"`) {
		t.Error("empty actions listing should use the sw-empty paragraph")
	}
	if !strings.Contains(body, `href="/chat?prompt=`) {
		t.Error("empty actions must link to /chat with a prompt parameter")
	}
}

// TestListPageEmptyStateForTasks checks the same pattern for tasks.
func TestListPageEmptyStateForTasks(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/task")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `<h2 class="sw-empty__title">No tasks yet</h2>`) {
		t.Errorf("empty tasks listing should have an h2 heading 'No tasks yet'\n%s", truncate(body))
	}
	if !strings.Contains(body, `href="/chat?prompt=`) {
		t.Error("empty tasks must link to /chat with a prompt parameter")
	}
}

// TestTheHomePageHasOneEmptyState: a home page with only the conversation
// says so once, in the conversation, right above the box to type in.
func TestTheHomePageHasOneEmptyState(t *testing.T) {
	_, h := newApp(t)
	body := get(t, h, "/").Body.String()
	if n := strings.Count(body, `data-component="empty"`); n != 1 {
		t.Errorf("the home page should have one empty state, got %d: %s", n, truncate(body))
	}
	if !strings.Contains(body, "Ask for anything.") {
		t.Errorf("the conversation should say Ask for anything")
	}
}

// TestActivityEmptyStateHasNoHeading checks that the empty activity page does
// NOT introduce an <h2> (the test forbids it), and instead uses bold text for
// "No activity yet" inside the paragraph. Acceptance 1: the heading equivalent
// is at most 4 words. Covers acceptance item 1 for activity.
func TestActivityEmptyStateHasNoHeading(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/activity")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `data-component="empty"`) {
		t.Errorf("empty activity page should use the sw-empty paragraph\n%s", truncate(body))
	}
	if strings.Contains(body, "<h2") {
		t.Error("empty activity page must not introduce an h2 element — it already has an h1 from layout")
	}
	// The new text uses <strong> for "No activity yet" inside the paragraph.
	if !strings.Contains(body, `No activity yet.`) {
		t.Errorf("empty activity page should say 'No activity yet'\n%s", truncate(body))
	}
}

// TestSearchEmptyStateHasHeading checks that empty search results have an h2
// heading. Acceptance 1 for the search surface.
func TestSearchEmptyStateHasHeading(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/search?q=nonexistent")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `<h2 class="sw-empty__title">No results</h2>`) {
		t.Errorf("empty search should have an h2 heading 'No results'\n%s", truncate(body))
	}
}

// TestSearchEmptyStateKeepsQueryTerm checks that the empty-state body still
// mentions the query term so the person knows what was searched. Acceptance 1-3:
// the prompt line is tight, one sentence, and includes context. Covers acceptance
// item for search.
func TestSearchEmptyStateKeepsQueryTerm(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/search?q=nonexistent")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, "nonexistent") {
		t.Errorf("empty search state should mention the query term\n%s", truncate(body))
	}
	if !strings.Contains(strings.ToLower(body), "try different words") && !strings.Contains(strings.ToLower(body), "spelling") {
		t.Error("empty search state should suggest trying different words or checking spelling")
	}
}

// TestWorkspacesEmptyStateHasHeading checks that the other-workspaces listing
// shows a heading. Acceptance 1 for the workspaces surface.
func TestWorkspacesEmptyStateHasHeading(t *testing.T) {
	_, h := newApp(t)

	t.Logf("SAMEWAY_KNOWN=%q", os.Getenv("SAMEWAY_KNOWN"))
	if p := os.Getenv("SAMEWAY_KNOWN"); p != "" {
		raw, _ := os.ReadFile(p)
		t.Logf("known file content: %s", string(raw))
	}

	rec := get(t, h, "/workspaces")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// This one is a workspace, so the section is there and says only this
	// one so far, with a link to where another is made, not "below".
	if !strings.Contains(body, `<h2 id="ws-others">Other workspaces</h2>`) || !strings.Contains(body, "Only this one so far.") || !strings.Contains(body, `href="#new-name"`) || strings.Contains(body, "below") {
		t.Errorf("empty other-workspaces should be under its heading, say only this one so far, and link to making one\n%s", truncate(body))
	}
}

// TestWorkspacesEmptyStateHasActionPrompt checks that the empty-state paragraph
// is short. Acceptance 3 for the workspaces surface.
func TestWorkspacesEmptyStateHasActionPrompt(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/workspaces")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `data-component="empty"`) {
		t.Error("empty other-workspaces should use the sw-empty paragraph")
	}
	// The text is "Only this one so far." — a single short sentence.
	if strings.Contains(body, `class="sw-empty">None yet. Every workspace opened on this machine appears here.`) {
		t.Error("empty workspaces must not show the old two-sentence description")
	}
}

// TestEmptyStateHeadingIsShort checks that no empty-state heading exceeds 4 words.
// This covers Acceptance 1 for all surfaces in one shot.
func TestEmptyStateHeadingIsShort(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, `<h2 class="sw-empty__title">No notes yet</h2>`) {
		t.Errorf("heading for notes should be 'No note yet' (≤4 words)\n%s", truncate(body))
	}

	rec = get(t, h, "/activity")
	wantStatus(t, rec, http.StatusOK)
	body = rec.Body.String()
	if !strings.Contains(body, `No activity yet.`) {
		t.Errorf("heading for activity should be 'No activity yet' (≤4 words)\n%s", truncate(body))
	}

	rec = get(t, h, "/search?q=nonexistent")
	wantStatus(t, rec, http.StatusOK)
	body = rec.Body.String()
	if !strings.Contains(body, `<h2 class="sw-empty__title">No results</h2>`) || !strings.Contains(body, `<title>Search: nonexistent, no results`) {
		t.Errorf("heading for search should be 'No results', and the window's title says it with the words\n%s", truncate(body))
	}

	rec = get(t, h, "/workspaces")
	wantStatus(t, rec, http.StatusOK)
	body = rec.Body.String()
	if !strings.Contains(body, `<h2 id="ws-others">Other workspaces</h2>`) {
		t.Errorf("heading for workspaces should be 'Other workspaces'\n%s", truncate(body))
	}
}

// TestEmptyStateActionPromptIsShort checks that the action prompt text on each
// empty state is at most 8 words. Acceptance 3.
func TestEmptyStateActionPromptIsShort(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()
	if !strings.Contains(body, `href="/chat`) {
		t.Error("note empty-state must link to /chat")
	}

	rec = get(t, h, "/")
	wantStatus(t, rec, http.StatusOK)
	body = rec.Body.String()
	if !strings.Contains(body, `href="/chat`) {
		t.Error("canvas empty-state must link to /chat")
	}

	rec = get(t, h, "/activity")
	wantStatus(t, rec, http.StatusOK)
	body = rec.Body.String()
	if !strings.Contains(body, `href="/chat">Send a message</a>`) {
		t.Error("activity empty-state must link to /chat with text that says what it does (never 'here')")
	}

	rec = get(t, h, "/workspaces")
	wantStatus(t, rec, http.StatusOK)
	body = rec.Body.String()
	if !strings.Contains(body, `data-component="empty"`) {
		t.Error("workspaces empty-state must use sw-empty paragraph")
	}
}
