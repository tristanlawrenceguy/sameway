package server_test

// Tests for task 0382: empty-state link includes ?prompt= parameter and
// chat page pre-fills textarea with it.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// TestEmptyStateLinkIncludesPromptParam checks that the empty notes listing
// page links to /chat with a prompt query parameter, so clicking navigates
// to the chat page with context for creating a note. Covers Acceptance 2.
func TestEmptyStateLinkIncludesPromptParam(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	// The empty-state link must include a prompt query parameter.
	if !strings.Contains(body, `href="/chat?prompt=`) {
		t.Errorf("empty-state link should include ?prompt= to pre-fill chat\n%s", truncate(body))
	}

	// The prompt text should say "Create a new note." for the notes type.
	if !strings.Contains(body, "Create+a+note.") && !strings.Contains(body, "Create%20a%20note.") {
		t.Errorf("prompt should instruct to create a note\n%s", truncate(body))
	}

	// It must NOT link to the dead /t/note/new route.
	if strings.Contains(body, `/t/note/new`) {
		t.Error("empty-state must not link to /t/note/new — that route returns 404")
	}
}

// TestEmptyStatePromptForAllTypes checks that every content type listing page
// puts a prompt parameter on the empty-state chat link, with text matching its
// own name. Covers Acceptance 2 for all types.
func TestEmptyStatePromptForAllTypes(t *testing.T) {
	_, h := newApp(t)

	cases := []struct {
		path string
		name string // what the prompt should say after "Create a new "
	}{
		{"/t/note", "note"},
		{"/t/action", "action"},
		{"/t/task", "task"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := get(t, h, c.path)
			wantStatus(t, rec, http.StatusOK)
			body := rec.Body.String()

			if !strings.Contains(body, `href="/chat?prompt=`) {
				t.Errorf("empty-state for %s should include ?prompt=\n%s", c.name, truncate(body))
			}

			expectedPrompt := "Create+a+" + c.name + "."
			expectedPromptEscaped := "Create%20a%20" + c.name + "."
			if !strings.Contains(body, expectedPrompt) && !strings.Contains(body, expectedPromptEscaped) {
				t.Errorf("prompt should say 'Create a %s.'\n%s", c.name, truncate(body))
			}

			deadRoute := "/t/" + c.name + "/new"
			if strings.Contains(body, deadRoute) {
				t.Errorf("empty-state must not link to %s — that route returns 404", deadRoute)
			}
		})
	}
}

// TestChatPagePreFillsFromPrompt checks that navigating to /chat?prompt=...
// puts the prompt text into the compose textarea's value. Covers Acceptance 2.
func TestChatPagePreFillsFromPrompt(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/chat?prompt="+url.PathEscape("Create a note."))
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, "Create a note.") {
		t.Errorf("chat page should pre-fill textarea with prompt text\n%s", truncate(body))
	}
}

// TestChatPagePromptOverridesAbout checks that when both ?prompt= and ?about=
// are present on the chat page, the prompt takes priority over about. Covers
// Acceptance 2 — a person coming from an empty-state link should see their
// prompt text, not a record context.
func TestChatPagePromptOverridesAbout(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/chat?prompt="+url.PathEscape("Create a note.")+"&about=note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, "Create a note.") {
		t.Errorf("chat page should show prompt text when both ?prompt= and ?about= are present\n%s", truncate(body))
	}

	// The about-based prefix must NOT appear.
	if strings.Contains(body, "About ") && strings.Contains(body, "(note):") {
		t.Error("when ?prompt= is given, the ?about= context should not appear in the compose value")
	}
}

// TestChatPageNoPromptMeansEmptyTextarea checks that visiting /chat with no
// prompt parameter leaves the textarea empty (no pre-filled text). This ensures
// the change doesn't break normal chat page usage. Covers Acceptance 2 — only
// explicit prompts should pre-fill.
func TestChatPageNoPromptMeansEmptyTextarea(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/chat")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if strings.Contains(body, `value="Create`) || strings.Contains(body, `value="About`) {
		t.Errorf("plain /chat must not pre-fill the textarea\n%s", truncate(body))
	}
}
