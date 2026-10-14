package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// An agent adding a block to a Home that already holds many is told how
// many, and that a tab of its own keeps what it tries apart; on a short
// page, nothing is said of it.
func TestAnAgentAddingToALongHomeIsToldSo(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	add := func(i int) []string {
		var out struct {
			Notes []string `json:"notes"`
		}
		json.Unmarshal(postJSON(t, h, http.MethodPost, "/api/block", map[string]any{"component": "heading", "props": map[string]any{"text": "Section " + string(rune('A'+i))}, "span": 12, "frame": "bare"}).Body.Bytes(), &out)
		return out.Notes
	}
	for i := 0; i < 5; i++ {
		if notes := add(i); len(notes) > 0 {
			t.Fatalf("a short page says nothing of it: %v", notes)
		}
	}
	var last []string
	for i := 5; i < 13; i++ {
		last = add(i)
	}
	if len(last) == 0 || !strings.Contains(last[len(last)-1], "Home, the page the person opens first, now holds") || !strings.Contains(last[len(last)-1], "a tab of its own") {
		t.Errorf("a long Home is said, with a tab of its own: %v", last)
	}
}
