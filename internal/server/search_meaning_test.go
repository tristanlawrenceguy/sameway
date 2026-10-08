package server_test

import (
	"context"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// With Ollama and an embedding model on this computer, a search finds what
// a record is about: the plumber finds the boiler repair, after any match
// by words, marked as found by meaning. Skipped where there is no model.
func TestASearchFindsByMeaning(t *testing.T) {
	a, _ := newApp(t)
	for _, title := range []string{"Boiler repair, call Marek", "Renew the passport", "Pancakes: flour, eggs, milk", "Holiday ideas: Lisbon in May"} {
		if _, err := a.Store.Create("note", map[string]any{"title": title}); err != nil {
			t.Fatal(err)
		}
	}
	h := server.New(a)
	if !h.RefreshMeaning(context.Background()) {
		t.Skip("no Ollama with an embedding model on this computer")
	}
	hits, _ := search.Matches(a.Store, a.Types, "the plumber")
	if len(hits) == 0 || hits[0].Title != "Boiler repair, call Marek" || !hits[0].Near {
		t.Fatalf("the plumber finds the boiler, by meaning: %+v", hits)
	}
	page := get(t, h, "/search?q=the+plumber").Body.String()
	if !strings.Contains(page, "Boiler repair, call Marek") {
		t.Errorf("the search page finds it too: %s", truncate(page))
	}
}
