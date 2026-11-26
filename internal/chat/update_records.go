package chat

import (
	"fmt"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A week left to pile up is twenty-seven late tasks. Asked to move them to
// next week, or to mark the ones done that are, a model made twenty-seven
// update_record calls, each a round trip and a place to stop half way, and
// the receipt was twenty-seven changes the person could only undo one at a
// time; more often it offered to and did not. The same change to several
// records is one call and one change: all are changed or none, and one Undo
// puts them all back.

func (s *Service) updateRecords(typeName string, ids, where []string, fields map[string]any) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if ids, err = s.named(t, ids, where); err != nil { // record_ops.go
		return fail("Nothing was changed: %v", err)
	}
	if len(ids) == 0 || len(fields) == 0 {
		return fail("pass ids or where, the records to change, and fields, what to change on each")
	}
	writes := make([]records.Writing, len(ids))
	for i, id := range ids {
		if _, err := s.Store.Get(t.Name, id); err != nil {
			return fail("Nothing was changed: ids[%d] is %s. Use find_records to get the ids, and call again with all of them.", i, records.NoRecord(t.Name, id))
		}
		own := make(map[string]any, len(fields)) // a write may fill in the map it is given
		for k, v := range fields {
			own[k] = v
		}
		writes[i] = records.Writing{ID: id, Fields: own}
	}
	_, done, i, err := records.WriteAll(s.Store, "updated", t.Name, writes) // all or none
	if err != nil {
		return fail("Nothing was changed, as %s %s could not be: %s Call again with that fixed.", t.Name, ids[i], fieldHelp(err, t))
	}
	c, titles := oneChange("updated", t, done)
	return toolResult{text: fmt.Sprintf("updated %s, as one change (one Undo puts them all back): %s.", schema.Count(len(ids), t.Name), schema.Names(titles)), change: &c}
}
