package server_test

import (
	"net/http"
	"strings"
	"testing"
)

// A collection as a board is a column per choice of a pick-list, in the
// order the field lists them, each saying how many it holds, every choice
// there even when empty; a type with no pick-list says so in words.
func TestACollectionAsABoardHasAColumnPerChoice(t *testing.T) {
	a, h := newApp(t)
	for _, p := range []map[string]any{
		{"title": "Garden", "status": "done"},
		{"title": "House", "status": "active"},
		{"title": "Shed", "status": "active"},
	} {
		if _, err := a.Store.Create("project", p); err != nil {
			t.Fatal(err)
		}
	}
	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "project", "as": "board", "label": "Projects"},
	}), http.StatusCreated)

	page := get(t, h, "/").Body.String()
	active := strings.Index(page, `Active <span class="sw-collection__count">(2)</span>`)
	done := strings.Index(page, `Done <span class="sw-collection__count">(1)</span>`)
	if active < 0 || done < 0 || active > done {
		t.Fatalf("the board has Active (2) then Done (1) as columns: %.2000s", page)
	}
	if g := strings.Index(page, "Garden"); g < done {
		t.Error("Garden is done, so it is in the Done column")
	}
	if i := strings.Index(page, ">House<"); i < active || i > done {
		t.Error("House is active, so it is in the Active column")
	}

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "task", "as": "board", "label": "Tasks"},
	}), http.StatusCreated)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, "a board needs a pick-list field for its columns, and task has none") {
		t.Error("a board of a type with no pick-list says why it cannot be shown")
	}
}
