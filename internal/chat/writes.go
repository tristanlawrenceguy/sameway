package chat

import (
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A change made through the API or on the command line is a change like
// any other: logged as the person's, with how it came and what the
// record was before, so it can be taken back. It used to go unlogged, so
// an agent told to tidy up could delete what nobody could restore.

// Through names a way in that is not a page, for the log's sentence:
// "You deleted note Plan, through the API".
const (
	ThroughAPI = "through the API"
	ThroughCLI = "through the command line"
)

// RecordWrite logs a create, update or delete of rec made through a way
// in. before is what the record was; nil for one just made.
func RecordWrite(st *store.Store, through, action string, rec *store.Record, before map[string]any) string {
	c := Change{Action: action, Component: rec.Type, ID: rec.ID, Before: before, Via: through}
	switch rec.Type {
	case BlockType:
		name, _ := rec.Fields["component"].(string)
		props, _ := rec.Fields["props"].(map[string]any)
		c.Component, c.Detail = name, Summarise(name, props)
		if action == "created" {
			c.Action = "added"
		} else if action == "deleted" {
			c.Action = "removed"
		}
		if action != "deleted" {
			c.Href = "/canvas/" + rec.ID
		}
	default:
		if t, ok := st.Types().Get(rec.Type); ok {
			c.Detail = recordTitle(t, rec)
			if !t.Internal && action != "deleted" {
				c.Href = "/t/" + rec.Type + "/" + rec.ID
			}
		}
	}
	return Record(st, "human", c)
}

// Imported is the batch an import from a file made: records that were not
// there before, so undoing it takes them away together.
func Imported(typ string, ids []string) map[string]any {
	items := make([]BatchItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, BatchItem{Type: typ, ID: id})
	}
	return Batch(items)
}
