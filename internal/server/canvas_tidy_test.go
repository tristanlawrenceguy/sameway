package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// named is the ids the page's tidy line sends.
func named(t *testing.T, page string) string {
	t.Helper()
	at := strings.Index(page, `name="ids" value="`)
	if at < 0 {
		t.Fatal("the tidy line sends what it named")
	}
	rest := page[at+len(`name="ids" value="`):]
	return rest[:strings.Index(rest, `"`)]
}

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
	if !strings.Contains(page, "2 blocks here cannot be shown or repeat another: “The weather is nice today.” is here twice; a button cannot be shown.") || !strings.Contains(page, "Tidy this page") {
		i := strings.Index(page, "here cannot")
		t.Fatalf("the tab offers to tidy, naming what goes:\n%s", page[max(0, i-100):min(len(page), i+300)])
	}
	postForm(t, h, "/tidy", url.Values{"canvas": {""}, "ids": {named(t, page)}})
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

// The press takes only what the line named, so a page left open while a
// block broke does not take that one too; a block that shows nothing yet
// is not broken, and is never offered; one block is said as one.
func TestTidyTakesOnlyWhatItNamed(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	a.Store.Create(records.BlockType, map[string]any{"component": "collection", "props": map[string]any{"type": "task"}, "position": 10, "canvas": ""})
	a.Store.Create(records.BlockType, map[string]any{"component": "button", "props": map[string]any{}, "position": 11, "canvas": ""})
	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, "1 block here cannot be shown or repeats another: a button cannot be shown.") {
		t.Fatalf("one broken block, said as one; the empty list is not offered:\n%s", truncate(page))
	}
	ids := named(t, page)
	late, _ := a.Store.Create(records.BlockType, map[string]any{"component": "button", "props": map[string]any{"variant": "secondary"}, "position": 12, "canvas": ""})
	res := postForm(t, h, "/tidy", url.Values{"canvas": {""}, "ids": {ids}})
	if _, err := a.Store.Get(records.BlockType, late.ID); err != nil {
		t.Error("a block broken after the line was read stays until it is named")
	}
	if _, err := a.Store.Get(records.BlockType, ids); err == nil {
		t.Error("the named block goes")
	}
	if told := after(t, h, res).Body.String(); !strings.Contains(told, "1 block is gone from this page.") {
		t.Errorf("the outcome says one as one:\n%s", truncate(told))
	}
}
