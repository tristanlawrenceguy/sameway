package server_test

import (
	"net/http"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
)

// TestDesignPageHasNoDuplicateIDs checks that the /design styleguide page has
// no duplicate id attributes. Before the fix, chart-caption repeated 6 times,
// chart-desc at least 3 times, and collection-task-title appeared multiple
// times because component templates render without a unique id prop on /design.
func TestDesignPageHasNoDuplicateIDs(t *testing.T) {
	_, h := newApp(t)

	var seen struct {
		Path    string
		Status  int
		Outline look.Outline
	}
	rec := get(t, h, "/api/look?path=/design")
	wantStatus(t, rec, http.StatusOK)
	decode(t, rec, &seen)

	// Assert that no "duplicate id" problems exist on /design.
	var dupIDs []string
	for _, p := range seen.Outline.Problems {
		if len(p) >= 14 && p[:14] == "duplicate id \"" {
			dupIDs = append(dupIDs, p)
		}
	}
	if len(dupIDs) > 0 {
		t.Errorf("/design should have no duplicate id attributes; found: %v", dupIDs)
	}
}
