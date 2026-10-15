package api_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// A block written says how the tab reads and its first three things to
// look at, with how many more and where they are; GET /api/layout says
// them all, with the arrangement that would fix them.
func TestABlockWriteSaysTheLayoutInBrief(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var last string
	for range 6 {
		var made struct{ Layout string }
		decode(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "heading", "props": map[string]any{"text": "Same"}, "span": 4}), &made)
		last = made.Layout

	}
	var whole struct {
		Reads    string
		ToLookAt []string `json:"to_look_at"`
		Arrange  string
	}
	decode(t, get(t, h, "/api/layout?canvas="), &whole)
	if !strings.HasPrefix(whole.Reads, "Layout now: row 1:") {
		t.Errorf("the whole says how the tab reads: %+v", whole)
	}
	if len(whole.ToLookAt) > 3 {
		if !strings.Contains(last, fmt.Sprintf("And %d more", len(whole.ToLookAt)-3)) || !strings.Contains(last, "GET /api/layout?canvas=") || strings.Contains(last, "(4)") {
			t.Errorf("the write says three, how many more and where: %s", last)
		}
	} else if strings.Contains(last, "GET /api/layout") {
		t.Errorf("three or fewer are said whole: %s", last)
	}
	if res := get(t, h, "/api/layout?canvas=nope"); res.Code != http.StatusNotFound {
		t.Errorf("a tab there is not: %d", res.Code)
	}
}
