package api_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// GET /api/block?canvas=<id> is that tab's blocks only, in the order the
// page reads them; canvas= alone is Home's; with no canvas, every block.
func TestOneTabsBlocksAreListed(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var tab struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/canvas", map[string]any{"name": "Garden"}), &tab)
	for i, label := range []string{"Second", "First"} {
		postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "canvas": tab.ID, "position": 2 - i, "props": map[string]any{"type": "task", "label": label}})
	}
	postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "collection", "props": map[string]any{"type": "note", "label": "Home notes"}})
	type listed struct {
		Records []struct {
			Fields map[string]any `json:"fields"`
		} `json:"records"`
	}
	labels := func(path string) []string {
		var l listed
		json.Unmarshal(get(t, h, path).Body.Bytes(), &l)
		var out []string
		for _, r := range l.Records {
			if p, ok := r.Fields["props"].(map[string]any); ok {
				if s, ok := p["label"].(string); ok {
					out = append(out, s)
				}
			}
		}
		return out
	}
	if got := labels("/api/block?canvas=" + tab.ID); len(got) != 2 || got[0] != "First" || got[1] != "Second" {
		t.Errorf("the Garden tab's blocks, in page order: %v", got)
	}
	home := labels("/api/block?canvas=")
	for _, l := range home {
		if l == "First" || l == "Second" {
			t.Errorf("canvas= is Home's blocks only: %v", home)
		}
	}
	if all := labels("/api/block"); len(all) < 3 {
		t.Errorf("with no canvas, every block: %v", all)
	}
}
