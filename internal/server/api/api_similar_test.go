package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// A block an agent adds over the API beside one that shows the same
// records says so in its layout, as the assistant's add does; the first
// says nothing of it, and neither does a change to it.
func TestAPIBlockSaysWhatAlreadyShowsTheSame(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	block := map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Tasks"}}
	var first, second struct {
		ID     string `json:"id"`
		Layout string `json:"layout"`
	}
	json.Unmarshal(postJSON(t, h, http.MethodPost, "/api/block", block).Body.Bytes(), &first)
	if first.ID == "" || strings.Contains(first.Layout, "same records") {
		t.Errorf("the first says nothing of it: %+v", first)
	}
	json.Unmarshal(postJSON(t, h, http.MethodPost, "/api/block", block).Body.Bytes(), &second)
	if !strings.HasPrefix(second.Layout, "This tab already had a collection of the same records: "+first.ID) || !strings.Contains(second.Layout, "Layout now") {
		t.Errorf("the second names the first, then the layout: %q", second.Layout)
	}
	changed := postJSON(t, h, http.MethodPatch, "/api/block/"+second.ID, map[string]any{"span": 6})
	if strings.Contains(changed.Body.String(), "same records") {
		t.Errorf("a change says nothing of it: %s", changed.Body)
	}
}
