package search_test

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// nearStore is a starter workspace whose search by meaning is a stand-in
// that always finds the boiler repair, so the order and the words said
// about it are checked without a model.
func nearStore(t *testing.T) (*store.Store, *schema.Set) {
	t.Helper()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(":memory:", types)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	boiler, _ := st.Create("note", map[string]any{"title": "Boiler repair, call Marek"})
	st.Create("note", map[string]any{"title": "Renew the passport"})
	st.Create("note", map[string]any{"title": "Ask the plumber about the tap"})
	search.UseMeaning(st, func(string) []string { return []string{"note/" + boiler.ID} })
	return st, types
}

// A hit found by meaning is marked, comes after what has the words, and
// the search says so in one sentence, the same one an agent reads.
func TestWhatIsCloseInMeaningIsSaidAndComesAfterTheWords(t *testing.T) {
	t.Parallel()
	st, types := nearStore(t)
	if !search.ByMeaning(st) {
		t.Fatal("a workspace using meaning says so")
	}
	r := search.Reader{Now: time.Now()}

	hits, some := search.Matches(st, types, "plumber", r)
	if len(hits) != 2 || hits[0].Near || !hits[1].Near || hits[1].Title != "Boiler repair, call Marek" {
		t.Fatalf("the word match first, then the one close in meaning: %+v", hits)
	}
	want := "First what has your words, then what is close in meaning."
	if got := search.Order(hits, some); got != want {
		t.Errorf("Order = %q, want %q", got, want)
	}
	if said := search.Narrow(hits, "plumber", "", 1).Said(); !strings.HasPrefix(said, want) {
		t.Errorf("an agent reads the same order: %q", said)
	}

	hits, some = search.Matches(st, types, "heating engineer", r)
	if want := "Nothing has all your words; these are close in meaning."; search.Order(hits, some) != want {
		t.Errorf("only by meaning: %q, %+v", search.Order(hits, some), hits)
	}

	hits, some = search.Matches(st, types, "the heating", r)
	if !some || len(hits) < 2 || !hits[0].Near {
		t.Fatalf("nothing has every word: by meaning first, then some words: %+v", hits)
	}
	if want := "Nothing has every word. First what is close in meaning, then what has some of the words."; search.Order(hits, some) != want {
		t.Errorf("by meaning, then some words: %q", search.Order(hits, some))
	}
}

// With no meaning, a search says what it always said: some words, or
// nothing about the order at all.
func TestOrderWithoutMeaningIsAsBefore(t *testing.T) {
	t.Parallel()
	hits := []search.Hit{{Type: "note", ID: "1"}}
	if got := search.Order(hits, true); got != search.SomeWords {
		t.Errorf("some words: %q", got)
	}
	if got := search.Order(hits, false); got != "" {
		t.Errorf("every word, nothing to say: %q", got)
	}
	if got := search.Order(nil, true); got != "" {
		t.Errorf("nothing found, nothing to say: %q", got)
	}
}
