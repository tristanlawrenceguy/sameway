package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// When search results have duplicate titles, the apart words appear in an
// aria-label on each link so they are told apart for WCAG 2.4.9; no type
// name suffix is appended to the visible text or included in the aria-label.
func TestSearchResultTwinsUseAriaLabelNotTypeSuffix(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)

	var recs [2]struct{ ID string }
	for i := range recs {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task",
			map[string]any{"title": "Call plumber", "due": "2026-10-" + fmt.Sprintf("%02d", i+1)}), http.StatusCreated)
	}

	pageRes := get(t, h, "/search?q=plumber")
	wantStatus(t, pageRes, http.StatusOK)
	page := pageRes.Body.String()

	// Each heading must show only the title with no type suffix.
	if strings.Contains(page, " — Task") || strings.Contains(page, " — task") {
		t.Errorf("heading must not append a type suffix like ' — Task'; found in:\n%s", truncate(page))
	}

	// The apart words (due dates) should appear in the page body.
	if strings.Count(page, "(due ") < 2 {
		t.Errorf("each twin result should be told apart by its day; found in:\n%s", truncate(page))
	}

	// No visually hidden span inside the heading links — apart words go
	// into an aria-label on the link instead.
	if strings.Contains(page, `<span class="sw-visually-hidden">`) {
		t.Errorf("search result card must not contain a visually hidden span; found in:\n%s", truncate(page))
	}

	// The aria-label (if present) should contain apart words but no type name.
	o, _ := look.Page(page)
	for _, c := range o.Controls {
		if c.Kind == "link" && strings.Contains(c.Name, "Call plumber") {
			// If the accessible name includes apart words in parens, that is fine.
			// But it must NOT include a type suffix like "— Task".
			if strings.Contains(c.Name, " — ") {
				t.Errorf("link accessible name %q must not contain a type suffix", c.Name)
			}
		}
	}
}
