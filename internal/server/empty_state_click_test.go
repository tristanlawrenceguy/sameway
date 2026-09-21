package server_test

// Tests for empty-state link navigation — clicking the <a href="/chat"> must
// actually navigate to /chat. These pin down that no script intercepts anchor
// clicks inside .sw-empty paragraphs, and that the rendered HTML is correct.
// Covers Acceptance 1-2 of task 0398 (backlog 0398).

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

// TestEmptyStateAnchorLinksToChat checks that an empty notes listing page
// renders an <a href="/chat"> inside the .sw-empty paragraph. This is the
// anchor a person clicks to create content via chat. Covers Acceptance 1.
func TestEmptyStateAnchorLinksToChat(t *testing.T) {
	_, h := newApp(t)

	rec := get(t, h, "/t/note")
	wantStatus(t, rec, http.StatusOK)
	body := rec.Body.String()

	if !strings.Contains(body, `<p class="sw-empty">`) {
		t.Errorf("empty-state must use <p class=\"sw-empty\">\n%s", truncate(rec.Body.String()))
	}
	if !strings.Contains(body, `href="/chat"`) {
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
	_, h := newApp(t)

	for _, typ := range []string{"note", "action", "task"} {
		t.Run(typ, func(t *testing.T) {
			rec := get(t, h, "/t/"+typ)
			wantStatus(t, rec, http.StatusOK)
			body := rec.Body.String()

			if !strings.Contains(body, `href="/chat"`) {
				t.Errorf("empty-state for %s must have href=\"/chat\"\n%s", typ, truncate(body))
			}

			deadRoute := "/t/" + typ + "/new"
			if strings.Contains(body, deadRoute) {
				t.Errorf("empty-state must not link to %s — that route returns 404", deadRoute)
			}
		})
	}
}

// TestNoJSScriptInterceptsEmptyStateAnchorClicks checks every JS file under
// design/base/ to confirm none install a document-level or <a> click handler
// that could prevent the empty-state link from navigating. This covers
// Acceptance 1-2: if a script intercepts clicks on anchors, navigation breaks.
func TestNoJSScriptInterceptsEmptyStateAnchorClicks(t *testing.T) {
	baseDir := "../../design/base"

	files, err := os.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("cannot read design/base/: %v", err)
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".js") {
			continue
		}
		data, err := os.ReadFile(baseDir + "/" + f.Name())
		if err != nil {
			t.Fatalf("cannot read %s: %v", f.Name(), err)
		}
		src := string(data)

		// Reject any handler that selects plain <a> elements — this can
		// intercept empty-state links if a click handler is attached.
		if strings.Contains(src, `querySelectorAll("a"`) {
			t.Errorf("%s: must not select plain <a> via querySelectorAll", f.Name())
		}
		if strings.Contains(src, "document.querySelector(\"a") {
			t.Errorf("%s: must not select plain <a> via document.querySelector", f.Name())
		}

		// Reject any document-level click handler that could catch anchor clicks.
		if strings.Contains(src, `document.addEventListener("click"`) ||
			strings.Contains(src, "document.addEventListener('click'") {
			t.Errorf("%s: must not install a document-level click listener", f.Name())
		}

		// Reject any handler that combines .sw-empty with <a> selection.
		if strings.Contains(src, ".sw-empty") {
			if strings.Contains(src, "addEventListener(\"click\"") ||
				strings.Contains(src, `querySelector("a"`) {
				t.Errorf("%s: must not attach handlers to or select <a> near .sw-empty", f.Name())
			}
		}
	}
}
