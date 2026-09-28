package server_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// A collection as a board is a column per choice of a pick-list, in the
// order the field lists them, each saying how many it holds, every choice
// there even when empty; a type with no pick-list says so in words.
func TestACollectionAsABoardHasAColumnPerChoice(t *testing.T) {
	a, h := newApp(t)
	ids := map[string]string{}
	for _, p := range []map[string]any{
		{"title": "Garden", "status": "done"},
		{"title": "House", "status": "active"},
		{"title": "Shed", "status": "active"},
	} {
		rec, err := a.Store.Create("project", p)
		if err != nil {
			t.Fatal(err)
		}
		ids[p["title"].(string)] = rec.ID
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

	if !strings.Contains(page, `class="sw-collection__tally"`) || !strings.Contains(page, `>Active (2)<span class="sw-visually-hidden">: Projects</span></a>`) {
		t.Error("a line over the board names each column with its count and leads to it")
	}
	if !strings.Contains(page, `Move<span class="sw-visually-hidden"> House</span></button>`) {
		t.Error("each card has a Move button named with the card")
	}

	// Move sends the field like any edit, comes back to the card and says
	// where it went, with an Undo.
	house := ids["House"]
	head, _, found := strings.Cut(page, "-"+house+`" tabindex="-1"`)
	if !found {
		t.Fatal("each card has an id to come back to")
	}
	at := head[strings.LastIndex(head, `id="`)+len(`id="`):] + "-" + house
	res := doWithReferer(t, h, "/t/project/"+house+"/props", url.Values{"prop-status": {"done"}, "back": {at}}, "http://example.com/")
	back, loc := landed(t, h, res)
	if loc != "/#"+at {
		t.Errorf("the move comes back to the card, %q, not %q", "/#"+at, loc)
	}
	next := back.Body.String()
	if !strings.Contains(next, "House moved from Active to Done.") || !strings.Contains(next, "Undo") {
		t.Errorf("the page says where House went, with an Undo: %.3000s", next)
	}
	if !strings.Contains(next, `Active <span class="sw-collection__count">(1)</span>`) {
		t.Error("Active now holds Shed alone")
	}
	bad := doWithReferer(t, h, "/t/project/"+house+"/props", url.Values{"prop-status": {"active"}, "back": {`x"><script>`}}, "http://example.com/")
	if loc := bad.Header().Get("Location"); strings.Contains(loc, "#") {
		t.Errorf("only an id is taken to come back to: %q", loc)
	}

	wantStatus(t, postJSON(t, h, http.MethodPost, "/api/block", map[string]any{
		"component": "collection", "props": map[string]any{"type": "entry", "as": "board", "label": "Entries"},
	}), http.StatusCreated)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, "a board needs a pick-list field for its columns, and entry has none") {
		t.Error("a board of a type with no pick-list says why it cannot be shown")
	}
}
