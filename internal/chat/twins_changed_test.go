package chat_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// A record changed that another open one is called the same as says so,
// with how to make them one; one with no twin says nothing of it.
func TestAChangeToATwinSaysSo(t *testing.T) {
	t.Parallel()
	svc := newFullService(t)
	a, _ := svc.Store.Create("event", map[string]any{"title": "Community Cleanup Day", "starts": "2026-10-22T09:00:00Z"})
	b, _ := svc.Store.Create("event", map[string]any{"title": "Community Cleanup Day", "starts": "2026-10-22T09:00:00Z"})
	raw, _ := json.Marshal(map[string]any{"type": "event", "id": a.ID, "fields": map[string]any{"where": "Riverside Park"}})
	text, _ := svc.Call("update_record", raw)
	if !strings.Contains(text, "is called the same as 1 other, still open: "+b.ID) || !strings.Contains(text, "keep_one") {
		t.Errorf("a change to a twin says so: %s", text)
	}
	lone, _ := svc.Store.Create("event", map[string]any{"title": "Picnic", "starts": "2026-10-23T09:00:00Z"})
	raw, _ = json.Marshal(map[string]any{"type": "event", "id": lone.ID, "fields": map[string]any{"where": "Park"}})
	if text, _ := svc.Call("update_record", raw); strings.Contains(text, "called the same") {
		t.Errorf("one with no twin says nothing of it: %s", text)
	}
}
