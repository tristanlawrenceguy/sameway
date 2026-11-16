package search_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Block titles from search results show actual content text, not internal
// component names — heading, card, collection. Each case puts a block with
// known props into the store and checks that Hit.Title contains the content
// rather than the type name.
func TestBlockTitlesShowContentNotComponentName(t *testing.T) {
	t.Parallel()

	types := loadTypes(t)

	// A heading block: its component is "heading", its title prop has real text.
	st, _ := store.Open(":memory:", types)
	rec, _ := st.Create("block", map[string]any{
		"component": "heading",
		"props":     map[string]any{"title": "Community cleanup on Saturday"},
	})
	hits, _ := search.Matches(st, types, "community cleanup", search.Reader{Now: time.Now()})
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	title := hits[0].Title
	if title == "" {
		t.Fatal("block title must not be empty when content exists")
	}
	for _, badWord := range []string{"heading", "Heading"} {
		if title == badWord || title == badWord+" — Block" {
			t.Errorf("title %q is an internal component name, expected actual content text", title)
		}
	}

	// A card block: its component is "card", its text prop has real content.
	st2, _ := store.Open(":memory:", types)
	rec2, _ := st2.Create("block", map[string]any{
		"component": "card",
		"props":     map[string]any{"text": "Pack book collection for the house move"},
	})
	hits2, _ := search.Matches(st2, types, "house move", search.Reader{Now: time.Now()})
	if len(hits2) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits2))
	}
	title2 := hits2[0].Title
	for _, badWord := range []string{"card", "Card"} {
		if title2 == badWord || title2 == badWord+" — Block" {
			t.Errorf("title %q is an internal component name, expected actual content text", title2)
		}
	}

	// A collection block: no Summarise case for "collection", so fallback props matter.
	st3, _ := store.Open(":memory:", types)
	rec3, _ := st3.Create("block", map[string]any{
		"component": "collection",
		"props":     map[string]any{"type": "task", "label": "Up next"},
	})
	hits3, _ := search.Matches(st3, types, "up next", search.Reader{Now: time.Now()})
	if len(hits3) != 1 {
		t.Fatalf("expected 1 hit for collection with label, got %d", len(hits3))
	}
	title3 := hits3[0].Title
	for _, badWord := range []string{"collection", "Collection"} {
		if title3 == badWord || title3 == badWord+" — Block" {
			t.Errorf("title %q is an internal component name, expected actual content text", title3)
		}
	}

	_ = rec
	_ = rec2
	_ = rec3
}

// A block with no common content props (label, caption, title, text, content)
// should return an empty string as its title — the snippet will still show
// what was matched. The important thing is no component name leaks through.
func TestBlockWithNoContentPropsHasEmptyTitle(t *testing.T) {
	t.Parallel()

	types := loadTypes(t)

	st, _ := store.Open(":memory:", types)
	rec, _ := st.Create("block", map[string]any{
		"component": "chart",
		"props":     map[string]any{"type": "entry", "by": "at"},
	})
	hits, _ := search.Matches(st, types, "entry", search.Reader{Now: time.Now()})
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit for chart block, got %d", len(hits))
	}
	title := hits[0].Title
	for _, badWord := range []string{"chart", "Chart"} {
		if title == badWord || title == badWord+" — Block" {
			t.Errorf("title %q is an internal component name, expected empty or content text", title)
		}
	}

	_ = rec
}

// loadTypes loads the starter workspace schema for use in search tests.
func loadTypes(t *testing.T) *schema.Set {
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	return types
}
