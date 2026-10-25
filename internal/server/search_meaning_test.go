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
	t.Parallel()
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
	hits, _ := search.Matches(a.Store, a.Types, "the plumber", search.Reader{Now: a.Now()})
	if len(hits) == 0 || hits[0].Title != "Boiler repair, call Marek" || !hits[0].Near {
		t.Fatalf("the plumber finds the boiler, by meaning: %+v", hits)
	}
	page := get(t, h, "/search?q=the+plumber").Body.String()
	if !strings.Contains(page, "Boiler repair, call Marek") {
		t.Errorf("the search page finds it too: %s", truncate(page))
	}
}

// The search page says why a result with none of the words is there, and
// in what order: the word match, then what is close in meaning, said in a
// sentence above them and on the result. The box says it finds by
// meaning. A stand-in finds the boiler repair, so no model is needed.
func TestTheSearchPageSaysWhatIsCloseInMeaning(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	boiler, _ := a.Store.Create("note", map[string]any{"title": "Boiler repair, call Marek"})
	a.Store.Create("note", map[string]any{"title": "Ask the plumber about the tap"})
	search.UseMeaning(a.Store, func(string) []string { return []string{"note/" + boiler.ID} })
	h := server.New(a)
	page := get(t, h, "/search?q=plumber").Body.String()
	for _, want := range []string{
		"First what has your words, then what is close in meaning.",
		"It also finds what is close in meaning.",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page says %q: %s", want, truncate(page))
		}
	}
	if !strings.Contains(page, `<p class="sw-card__meta">Note`) && !strings.Contains(page, `<p class="sw-card__meta">note`) {
		t.Error("type info should appear in a separate meta paragraph below the heading")
	}
	if strings.Index(page, "Ask the plumber") > strings.Index(page, "Boiler repair") {
		t.Error("the word match comes first")
	}
}

// Help says how far the model has come while it is fetched, so a 274 MB
// wait is not a silence.
func TestHelpSaysHowFarSearchByMeaningHasCome(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	h := server.New(a)
	h.MeaningFetching(0, 0)
	if got := h.MeaningLine(); !strings.Contains(got, "fetching nomic-embed-text, 274 MB.") {
		t.Errorf("before Ollama says the size: %q", got)
	}
	h.MeaningFetching(137, 274)
	if got := h.MeaningLine(); !strings.Contains(got, "fetching 50% of nomic-embed-text, 274 MB.") {
		t.Errorf("half way: %q", got)
	}
}

// How far search by meaning has come is the workspace's: a second server
// on it says the same, and another workspace open beside it says nothing
// of it.
func TestSearchByMeaningIsEachWorkspacesOwn(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	server.New(a).MeaningFetching(137, 274)
	if got := server.New(a).MeaningLine(); !strings.Contains(got, "fetching 50%") {
		t.Errorf("a second server on the workspace says how far it has come: %q", got)
	}
	b, _ := newApp(t)
	if got := server.New(b).MeaningLine(); strings.Contains(got, "fetching") {
		t.Errorf("another workspace is not fetching: %q", got)
	}
}
