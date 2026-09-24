package server_test

import (
	"html"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A field with a fixed set of values carries them, by name, for the
// inline editor to offer as a list: the habit an entry is for by its
// title, never its id, and an enum by its values.
func TestAFieldWithChoicesOffersThemByName(t *testing.T) {
	a, h := newApp(t)
	water, _ := a.Store.Create(server.HabitType, map[string]any{"name": "Water", "cadence": "week", "aim": "limit"})
	a.Store.Create(server.HabitType, map[string]any{"name": "Stretch"})
	entry, _ := a.Store.Create(server.EntryType, map[string]any{"habit": water.ID, "at": when.Store(time.Now(), false), "amount": 2})

	page := html.UnescapeString(get(t, h, "/t/entry/"+entry.ID).Body.String())
	for _, want := range []string{`data-prop="habit" data-source="` + water.ID + `" data-options=`, `{"value":"` + water.ID + `","label":"Water"}`, `"label":"Stretch"}`} {
		if !strings.Contains(page, want) {
			t.Errorf("the habit is chosen by name, missing %q", want)
		}
	}
	page = html.UnescapeString(get(t, h, "/t/habit/"+water.ID).Body.String())
	// An enum offers its values by the names the schema gives them, and
	// stores the value.
	if !strings.Contains(page, `[{"value":"reach","label":"At least the target"},{"value":"limit","label":"At most the target"},{"value":"record","label":"Just keep a record"}]`) {
		t.Error("an enum offers its values by name")
	}
}
