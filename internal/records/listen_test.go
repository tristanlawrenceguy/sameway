package records

import (
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A store's listeners are its own: a change in one workspace is not heard
// by another open beside it, and they go when the store closes.
func TestListenersAreOneStoresAndGoOnClose(t *testing.T) {
	t.Parallel()
	types, err := schema.Load(filepath.Join("..", "..", "examples", "workspaces", "starter", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	open := func() *store.Store {
		st, err := store.Open(":memory:", types)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { st.Close() })
		return st
	}
	a, b := open(), open()
	heardA, heardB := 0, 0
	Listen(a, func(string, Change) { heardA++ })
	Listen(b, func(string, Change) { heardB++ })
	if _, _, err := WriteAs(b, Who{Actor: "human"}, "created", "note", "", map[string]any{"title": "In b"}); err != nil {
		t.Fatal(err)
	}
	if heardA != 0 || heardB != 1 {
		t.Errorf("a change in b was heard %d times by a and %d by b, want 0 and 1", heardA, heardB)
	}
	a.Close()
	if a.Kept(feedKey{}) != nil {
		t.Error("a closed store's listeners are still kept")
	}
}
