package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A new workspace opens on its conversation even when an agent adds the
// first block before anyone has opened Home, as a new tab does: the chat
// used to be put on a tab only when the tab was opened empty, so a block
// added first left the tab with no chat at all.
func TestFirstBlockByAPIKeepsTheChat(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	var garden struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/canvas", map[string]any{"name": "Garden"}), &garden)
	for _, tab := range []struct{ canvas, path string }{{"", "/"}, {garden.ID, "/c/" + garden.ID}} {
		wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
			"component": "collection", "canvas": tab.canvas, "props": map[string]any{"type": "task", "label": "First"}}), http.StatusCreated)
		page := get(t, h, tab.path).Body.String()
		if !strings.Contains(page, `data-block-component="chat"`) || !strings.Contains(page, "First") {
			t.Errorf("%s: the first block an agent adds comes with the conversation, not instead of it", tab.path)
		}
	}
}
