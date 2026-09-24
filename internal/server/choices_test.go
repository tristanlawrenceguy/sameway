package server_test

import (
	"encoding/json"
	"html"
	"regexp"
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

	// Every place the page offers the habit to the editor, it offers each
	// habit by its name, with the one there now as its value.
	page := get(t, h, "/t/entry/"+entry.ID).Body.String()
	habits := offered(t, page, "habit")
	if habits[water.ID] != "Water" || !containsLabel(habits, "Stretch") || !strings.Contains(html.UnescapeString(page), `data-prop="habit" data-source="`+water.ID+`"`) {
		t.Errorf("the habit is chosen by name: %v", habits)
	}
	// An enum offers its values by the names the schema gives them, and
	// stores the value.
	aims := offered(t, get(t, h, "/t/habit/"+water.ID).Body.String(), "aim")
	if aims["reach"] != "At least the target" || aims["limit"] != "At most the target" || aims["record"] != "Just keep a record" {
		t.Errorf("an enum offers its values by name: %v", aims)
	}
}

// offered reads every data-options a page carries for one field, as value
// to label; the order of keys inside each option does not matter.
func offered(t *testing.T, page, prop string) map[string]string {
	t.Helper()
	out := map[string]string{}
	re := regexp.MustCompile(`data-prop="` + prop + `"[^>]*?data-options="([^"]*)"`)
	for _, m := range re.FindAllStringSubmatch(page, -1) {
		var list []struct{ Value, Label string }
		if err := json.Unmarshal([]byte(html.UnescapeString(m[1])), &list); err != nil {
			t.Fatalf("data-options for %s is not a list of choices: %v", prop, err)
		}
		for _, c := range list {
			out[c.Value] = c.Label
		}
	}
	if len(out) == 0 {
		t.Fatalf("the page offers no choices for %s", prop)
	}
	return out
}

func containsLabel(m map[string]string, label string) bool {
	for _, l := range m {
		if l == label {
			return true
		}
	}
	return false
}
