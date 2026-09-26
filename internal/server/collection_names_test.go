package server_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// Two lists of one type on a canvas are each named by their own heading,
// and a list cut short by its limit says so and says its link has them all.
func TestTwoListsOfATypeAreNamedByTheirOwnHeadings(t *testing.T) {
	_, h := newApp(t)
	for _, title := range []string{"Order compost", "Dig the pond", "Plant garlic"} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": title}), http.StatusCreated)
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Soon"}}), http.StatusCreated)
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Two of them", "limit": 2}}), http.StatusCreated)
	page := get(t, h, "/").Body.String()
	if n := strings.Count(page, `id="collection-task-title"`); n != 0 {
		t.Errorf("each list's heading has an id from its block, not its type; %d share one", n)
	}
	for _, label := range []string{"Soon", "Two of them"} {
		m := regexp.MustCompile(`<h\d class="sw-collection__title" id="([^"]+)">` + label + `<`).FindStringSubmatch(page)
		if m == nil {
			t.Fatalf("no heading %q", label)
		}
		id := m[1]
		if strings.Count(page, `aria-labelledby="`+id+`"`) != 1 || strings.Count(page, `id="`+id+`"`) != 1 {
			t.Errorf("the list %q is named by its own heading, %q, and only it", label, id)
		}
	}
	if !strings.Contains(page, "Showing the first 2.") || !strings.Contains(page, "See all of them") {
		t.Errorf("a list cut short by its limit says so\n%s", truncate(page))
	}
	if strings.Count(page, "See the list") != 1 {
		t.Errorf("a list that shows every match keeps See the list")
	}
}
