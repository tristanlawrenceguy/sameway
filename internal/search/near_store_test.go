package search

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Finding by meaning is one workspace's: another open beside it finds by
// word alone, and it goes when the store closes.
func TestMeaningIsOneStoresAndGoesOnClose(t *testing.T) {
	t.Parallel()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	open := func() (*store.Store, string) {
		st, err := store.Open(":memory:", types)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { st.Close() })
		rec, err := st.Create("note", map[string]any{"title": "Boiler repair, call Marek"})
		if err != nil {
			t.Fatal(err)
		}
		return st, "note/" + rec.ID
	}
	a, keyA := open()
	b, _ := open()
	UseMeaning(a, func(string) []string { return []string{keyA} })
	if hits, _ := Matches(a, types, "plumber", Reader{Now: time.Now()}); len(hits) != 1 || !hits[0].Near {
		t.Errorf("a finds the boiler repair by meaning, got %+v", hits)
	}
	if hits, _ := Matches(b, types, "plumber", Reader{Now: time.Now()}); len(hits) != 0 {
		t.Errorf("b has no meaning of its own and finds nothing, got %+v", hits)
	}
	UseMeaning(b, nil)
	if ByMeaning(b) {
		t.Error("UseMeaning(nil) leaves a workspace saying it finds by meaning")
	}
	a.Close()
	if nearFinder(a) != nil || ByMeaning(a) {
		t.Error("a closed store keeps its way of finding by meaning")
	}
}
