package blocks

import (
	"maps"
	"net/url"
	"slices"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// What a collection is set up to show, in words, and the form that keeps
// a person's choices as its setup (server/collection_keep.go keeps them).

// keepForm is what Keep these choices sends: the picks in the address
// that belong to this block, and the page to come back to.
func keepForm(q url.Values, block, back string) map[string]any {
	prefix := "c-" + block + "-"
	fields := []any{}
	for _, name := range slices.Sorted(maps.Keys(q)) {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		for _, v := range q[name] {
			fields = append(fields, map[string]any{"name": name, "value": v})
		}
	}
	fields = append(fields, map[string]any{"name": "from", "value": back})
	return map[string]any{"action": "/canvas/" + block + "/keep", "fields": fields}
}

// setupWords is what a collection is set up to show, said after how many
// match when no choice is made: "9 tasks, not done, due soonest first".
// Once choices are kept this is where they are read back.
func setupWords(t *schema.Type, where []string, order string, choices []choice) string {
	var said []string
	if w := query.Words(t, where); w != "" {
		said = append(said, w)
	}
	if order != "" && order != baseOrder("") {
		words := strings.TrimPrefix(OrderWords(t, order), ", ")
		if len(choices) > 0 {
			for _, o := range choices[0].options {
				if o.value == order && o.label != "As set up" {
					words = lowerFirst(o.label)
				}
			}
		}
		said = append(said, words)
	}
	return strings.Join(said, ", ")
}

// countSaying is how many match, with what the list is set up to show.
func countSaying(t *schema.Type, n int, setup string) string {
	if n == 0 {
		return ""
	}
	if setup == "" {
		return schema.Count(n, t.Name)
	}
	return schema.Count(n, t.Name) + ", " + setup
}

// Kept is what a person picked on a collection, read from picks as its
// page reads them, so only what the choices offer can be kept: the where
// it would then have, its order when that differs from how it was set up
// ("" when not), and the picks in words. ok is false when nothing picked
// differs from the setup.
func Kept(t *schema.Type, props map[string]any, block string, picks url.Values) (where []string, order, said string, ok bool) {
	where = Strs(props["where"])
	base, _ := props["order"].(string)
	by := ""
	if f, err := BoardField(t, props["by"]); err == nil && props["as"] == "board" {
		by = f.Name
	}
	out := map[string]any{"id": "collection-" + block}
	kept, sorted, active := applyChoices(out, collectionChoices(t, where, base, by), &Page{Path: "/", Query: picks}, block, where, base)
	if !active {
		return nil, "", "", false
	}
	said, _ = out["choices"].(map[string]any)["showing"].(string)
	if sorted != baseOrder(base) {
		order = sorted
	}
	return kept, order, said, true
}
