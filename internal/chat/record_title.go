package chat

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// recordTitle is what a record is called: its title field, or its id. An
// entry has no title of its own, so it is called by its habit and how
// much, the same as on its page: "Read: 25 minutes", not its id.
func recordTitle(st *store.Store, t *schema.Type, rec *store.Record) string {
	if t.Name == EntryType {
		habit, unit := "", ""
		if id, _ := rec.Fields["habit"].(string); id != "" {
			ht, ok := st.Types().Get("habit")
			if h, err := st.Get("habit", id); ok && err == nil {
				habit = recordTitle(st, ht, h)
				unit, _ = h.Fields["unit"].(string)
			}
		}
		return track.EntryName(habit, unit, rec.Fields)
	}
	if v, ok := rec.Fields[t.Title].(string); ok && strings.TrimSpace(v) != "" {
		return trim.Title(v)
	}
	return rec.ID
}
