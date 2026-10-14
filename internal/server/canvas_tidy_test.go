package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A tab with a block that cannot be shown and a block the same as another
// offers the person to tidy it; the press takes those away, and only
// those, as one change Undo puts back; a tidy tab offers nothing.
func TestATabCanBeTidiedOfWhatCannotBeShownOrRepeats(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	text := map[string]any{"content": "The weather is nice today."}
	a.Store.Create(records.BlockType, map[string]any{"component": "text", "props": text, "position": 10, "canvas": ""})
	a.Store.Create(records.BlockType, map[string]any{"component": "text", "props": text, "position": 11, "canvas": ""})
	a.Store.Create(records.BlockType, map[string]any{"component": "button", "props": map[string]any{}, "position": 12, "canvas": ""})
	before, _ := a.Store.List(records.BlockType, store.ListOptions{})

	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, "2 blocks here cannot be shown or repeat another.") || !strings.Contains(page, "Tidy this page") {
		t.Fatalf("the tab offers to tidy:\n%s", truncate(page))
	}
	postForm(t, h, "/tidy", url.Values{"canvas": {""}})
	after, _ := a.Store.List(records.BlockType, store.ListOptions{})
	if len(after) != len(before)-2 {
		t.Errorf("the broken one and the repeat go, the first stays: %d of %d left", len(after), len(before))
	}
	if strings.Contains(get(t, h, "/").Body.String(), "Tidy this page") {
		t.Error("a tidy tab offers nothing")
	}
	postForm(t, h, "/activity/"+newestEntry(t, a.Store).ID+"/undo", nil)
	if back, _ := a.Store.List(records.BlockType, store.ListOptions{}); len(back) != len(before) {
		t.Errorf("one Undo puts them back: %d of %d", len(back), len(before))
	}
}
