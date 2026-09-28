package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/peers"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func keepInStep(t *testing.T, a *app.App, ha http.Handler, b *app.App) {
	t.Helper()
	srv := httptest.NewServer(ha)
	defer srv.Close()
	if _, err := peers.With(context.Background(), http.DefaultClient, b.Store, strings.TrimPrefix(srv.URL, "http://")); err != nil {
		t.Fatal(err)
	}
}

// A content type made on one computer, and what is kept in it, reach the
// other: its file in schema/, its table, its records, one pointing at
// another type that came in the same exchange.
func TestANewTypeTravelsWithItsRecords(t *testing.T) {
	a, ha := newApp(t)
	b, _ := newApp(t)
	if _, err := a.AddType(&schema.Type{Name: "ingredient", Fields: []schema.Field{{Name: "name", Type: "string", Required: true}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := a.AddType(&schema.Type{Name: "recipe", Fields: []schema.Field{{Name: "name", Type: "string", Required: true}, {Name: "main", Type: "ref", To: "ingredient"}}}); err != nil {
		t.Fatal(err)
	}
	egg, _ := a.Store.Create("ingredient", map[string]any{"name": "Egg"})
	omelette, err := a.Store.Create("recipe", map[string]any{"name": "Omelette", "main": egg.ID})
	if err != nil {
		t.Fatal(err)
	}

	keepInStep(t, a, ha, b)
	if _, ok := b.Types.Get("recipe"); !ok {
		t.Fatal("the recipe type should reach the other computer")
	}
	if _, err := os.Stat(filepath.Join(b.Workspace.Dir, "schema", "recipe.yaml")); err != nil {
		t.Errorf("the type is written to schema/ there too: %v", err)
	}
	got, err := b.Store.Get("recipe", omelette.ID)
	if err != nil || got.Fields["name"] != "Omelette" || got.Fields["main"] != egg.ID {
		t.Errorf("the recipe comes with its type, pointing at its ingredient: %v %v", got, err)
	}
}

// Two computers adding different fields to notes both keep both, with
// what is written in them.
func TestFieldsAddedOnTwoComputersBothStay(t *testing.T) {
	a, ha := newApp(t)
	b, hb := newApp(t)
	keepInStep(t, a, ha, b)
	if _, err := a.AddField("note", schema.Field{Name: "rating", Type: "int"}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.AddField("note", schema.Field{Name: "mood", Type: "string"}); err != nil {
		t.Fatal(err)
	}
	n, _ := a.Store.Create("note", map[string]any{"title": "Trip", "rating": 5})
	keepInStep(t, a, ha, b)
	keepInStep(t, b, hb, a)
	for name, x := range map[string]*app.App{"a": a, "b": b} {
		note, _ := x.Types.Get("note")
		_, r := note.Field("rating")
		_, m := note.Field("mood")
		if !r || !m {
			t.Errorf("%s should have both fields: rating %v, mood %v", name, r, m)
		}
	}
	if got, err := b.Store.Get("note", n.ID); err != nil || got.Fields["rating"] != int64(5) && got.Fields["rating"] != 5 && got.Fields["rating"] != float64(5) {
		t.Errorf("what was written in the new field arrives with it: %v %v", got, err)
	}
}

// A record that arrives before its type waits, and is there once the type
// comes, whatever order the two arrive in.
func TestARecordWaitsForItsType(t *testing.T) {
	a, _ := newApp(t)
	b, _ := newApp(t)
	a.AddType(&schema.Type{Name: "plant", Fields: []schema.Field{{Name: "name", Type: "string"}}})
	p, _ := a.Store.Create("plant", map[string]any{"name": "Basil"})
	all, _ := a.Store.Since(map[string]string{})
	var records, types []store.Stamp
	for _, st := range all {
		if st.Type == store.SchemaType {
			types = append(types, st)
		} else {
			records = append(records, st)
		}
	}
	b.Store.Apply(records)
	if _, err := b.Store.Get("plant", p.ID); err == nil {
		t.Fatal("there is no plant type here yet")
	}
	b.Store.Apply(types)
	if got, err := b.Store.Get("plant", p.ID); err != nil || got.Fields["name"] != "Basil" {
		t.Errorf("once the type comes, the plant that waited is there: %v %v", got, err)
	}
}
