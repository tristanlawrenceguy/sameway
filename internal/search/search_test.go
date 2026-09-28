package search_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// One search finds what a person has by the words in it: records of every
// type and blocks on the canvas, title matches first, each with the words
// around the match and its page; history stays out of it.
func TestFindLooksThroughEverythingAPersonHas(t *testing.T) {
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	st.Create("note", map[string]any{"title": "Call the plumber", "body": "About the kitchen tap. Ask for a quote first."})
	st.Create("note", map[string]any{"title": "Garden", "body": "The plumber said the outside tap needs a new washer, plumber comes Tuesday."})
	st.Create("note", map[string]any{"title": "Reading list", "body": "Nothing to do with pipes."})
	st.Create("block", map[string]any{"component": "list", "props": map[string]any{"items": []any{"Milk", "Call plumber back"}}})
	st.Create("message", map[string]any{"role": "user", "content": "plumber plumber plumber"})
	st.Create("activity", map[string]any{"summary": "You said plumber", "actor": "human", "action": "said"})

	hits := search.Find(st, types, "plumber")
	if len(hits) != 3 {
		t.Fatalf("three things mention the plumber, got %d: %+v", len(hits), hits)
	}
	if hits[0].Title != "Call the plumber" || hits[0].Type != "note" || !strings.HasPrefix(hits[0].Href, "/t/note/") {
		t.Errorf("a title match comes first with its page, got %+v", hits[0])
	}
	var block *search.Hit
	for i := range hits {
		if hits[i].Type == "block" {
			block = &hits[i]
		}
	}
	if block == nil || !strings.HasPrefix(block.Href, "/canvas/") || !strings.Contains(block.Snippet, "Call plumber back") {
		t.Errorf("a canvas block is found by its words and leads to its page, got %+v", block)
	}
	for _, h := range hits {
		if h.Title == "Garden" && !strings.Contains(h.Snippet, "plumber") {
			t.Errorf("the snippet should show the words around the match, got %q", h.Snippet)
		}
	}

	// Every word must appear; case does not matter; blank finds nothing.
	if two := search.Find(st, types, "Plumber Tuesday"); len(two) != 1 || two[0].Title != "Garden" {
		t.Errorf("all words must match, got %+v", two)
	}
	if none := search.Find(st, types, "   "); none != nil {
		t.Errorf("a blank search finds nothing, got %+v", none)
	}
}
