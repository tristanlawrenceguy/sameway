package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/query"
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

var updateRecordsOp = Op{Title: "Change several records", Traits: Traits{Idempotent: true},
	Words: []string{"all of", "each", "every", "these", "move them", "mark them"},
	Doing: func(a callArgs) string {
		if len(a.IDs) == 0 {
			return "Updating " + or(schema.Plural(a.Type), "records")
		}
		return "Updating " + schema.Count(len(a.IDs), a.Type)
	},
	Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
		return s.updateRecords(a.Type, a.IDs, a.Where, a.Fields)
	},
	Tool: llm.Tool{Name: "update_records", Description: "Make the same change to several records of one content type in one change: move the late tasks to next week, mark these done, tag them all. Name them by ids, or by where to change every one it matches without finding them first (the answer names each). All are changed or none: if one cannot be, nothing is changed and you are told which. One Undo puts them all back.",
		Schema: obj(map[string]any{
			"type":   typeArg,
			"ids":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "The records to change, from find_records."},
			"where":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "In place of ids: change every record these conditions match, as find_records takes them."},
			"fields": map[string]any{"type": "object", "description": "The fields to change on every one of them, and their new values, as update_record takes."},
		}, "type", "fields")}, Offered: typed}

func (s *Service) updateRecords(typeName string, ids, where []string, fields map[string]any) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if len(ids) == 0 && len(where) > 0 { // found and changed in one step, so none is missed between them
		recs, err := query.Filter(s.Store, t, where, "", 0, s.Now())
		if err != nil {
			return fail("Nothing was changed: %v", err)
		}
		if len(recs) == 0 {
			return fail("Nothing was changed: no %s matches %s.", t.Name, strings.Join(where, ", "))
		}
		for _, r := range recs {
			ids = append(ids, r.ID)
		}
	}
	ids = unique(ids) // remove.go: a repeat is one record, changed once
	if len(ids) == 0 || len(fields) == 0 {
		return fail("pass ids or where, the records to change, and fields, what to change on each")
	}
	writes := make([]records.Writing, len(ids))
	for i, id := range ids {
		if _, err := s.Store.Get(t.Name, id); err != nil {
			return fail("Nothing was changed: ids[%d] is %s. Use find_records to get the ids, and call update_records again with all of them.", i, records.NoRecord(t.Name, id))
		}
		own := make(map[string]any, len(fields)) // a write may fill in the map it is given
		for k, v := range fields {
			own[k] = v
		}
		writes[i] = records.Writing{ID: id, Fields: own}
	}
	_, done, i, err := records.WriteAll(s.Store, "updated", t.Name, writes) // all or none
	if err != nil {
		return fail("Nothing was changed: %s %s could not be — %s. %s Fix the fields and call update_records again.", t.Name, ids[i], humanizeValidationError(err.Error()), fieldHelp(err, t))
	}
	ops, titles := records.Joined(done), make([]string, len(done))
	for i, c := range done {
		titles[i] = c.Detail
	}
	c := records.Change{Action: "updated", Component: t.Name, Href: "/t/" + t.Name, Ops: ops,
		Detail: schema.Count(len(ids), t.Name) + ": " + listed(titles, 4)} // remove.go
	if len(ids) == 1 {
		c.ID, c.Detail, c.Href = ids[0], titles[0], "/t/"+t.Name+"/"+ids[0]
	}
	return toolResult{text: fmt.Sprintf("updated %s, as one change (one Undo puts them all back): %s.", schema.Count(len(ids), t.Name), schema.Names(titles)), change: &c}
}
