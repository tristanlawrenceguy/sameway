package server_test

// Tests for empty-state link navigation — clicking the <a href="/chat"> must
// actually navigate to /chat. These pin down that no script intercepts anchor
// clicks inside .sw-empty paragraphs, and that the rendered HTML is correct.
// Covers Acceptance 1-2 of task 0398 (backlog 0398).

import (
	"net/http"
	"strings"
	"testing"
)

// TestEmptyStateAnchorLinksToChat checks that an empty notes listing page
// renders an <a href="/chat"> inside the .sw-empty paragraph. This is the
// anchor a person clicks to create content via chat. Covers Acceptance 1.
func TestEmptyStateAnchorLinksToChat(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `data-component="empty"`) {
		t.Errorf("empty-state must use the empty component\n%s", truncate(rec.Body.String()))
	}
	if !strings.Contains(body, `href="/chat`) {
		t.Errorf("empty-state anchor must have href=\"/chat\" so clicking navigates there\n%s", truncate(body))
	}

	// It must NOT link to any dead form route.
	if strings.Contains(body, `/t/note/new`) {
		t.Error("empty-state must not link to /t/note/new — that route returns 404")
	}
}

// TestEmptyStateAnchorNavigatesForAllTypes checks that the empty-state link
// points to /chat on every content type listing page, not a dead form route.
// Covers Acceptance 2 (all content type list pages).
func TestEmptyStateAnchorNavigatesForAllTypes(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	for _, typ := range []string{"note", "action", "task"} {
		t.Run(typ, func(t *testing.T) {
			rec := get(t, h, "/t/"+typ)
			wantStatus(t, rec, http.StatusOK)
			body := rec.Body.String()

			if !strings.Contains(body, `href="/chat`) {
				t.Errorf("empty-state for %s must have href=\"/chat\"\n%s", typ, truncate(body))
			}

			deadRoute := "/t/" + typ + "/new"
			if strings.Contains(body, deadRoute) {
				t.Errorf("empty-state must not link to %s — that route returns 404", deadRoute)
			}
		})
	}
}

// That no script stops the empty state's link from going where it says is
// checked by pressing it in a browser: behave-styles.mjs.
