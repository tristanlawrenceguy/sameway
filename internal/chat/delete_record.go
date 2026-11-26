package chat

import (
	"fmt"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Deleting records. With no tool for it, a person who asked to delete a
// note was told how to do it themselves, and asked "shall I delete it?"
// by an assistant that could not. A deleted record is kept in the log
// with everything it had, so one Undo puts it back: it is reversible, and
// is done as asked, as a block taken off a page is. What the person did
// not ask to delete is asked first, with propose_change.

var deleteRecordOp = Op{Title: "Delete records", Traits: Traits{Destructive: true, Idempotent: true},
	Doing: func(a callArgs) string {
		if len(a.IDs) > 1 {
			return "Deleting " + schema.Count(len(a.IDs), a.Type)
		}
		return "Deleting" + or(an(schema.Words(a.Type)), " a record")
	},
	Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
		ids := a.IDs
		if a.ID != "" {
			ids = append([]string{a.ID}, ids...)
		}
		return s.deleteRecords(a.Type, ids, a.Where)
	},
	Tool: llm.Tool{Name: "delete_record", Description: "Delete records the person asked to delete: one by id, several by ids, or every one where matches. All or none, as one change: one Undo puts them all back with everything they had.",
		Schema: obj(map[string]any{
			"type":  typeArg,
			"id":    map[string]any{"type": "string", "description": "From find_records, or a page's /t/<type>/<id>."},
			"ids":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"where": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "As find_records takes it."},
		}, "type")}, Offered: typed}

func (s *Service) deleteRecords(typeName string, ids, where []string) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	if ids, err = s.named(t, ids, where); err != nil { // record_ops.go
		return fail("Nothing was deleted: %v", err)
	}
	if len(ids) == 0 {
		return fail("Nothing was deleted: name the record by id, several by ids, or every one where matches.")
	}
	writes := make([]records.Writing, len(ids))
	for i, id := range ids {
		if _, err := s.Store.Get(t.Name, id); err != nil {
			return fail("Nothing was deleted: %s. find_records gives the ids.", records.NoRecord(t.Name, id))
		}
		writes[i] = records.Writing{ID: id}
	}
	_, done, i, err := records.WriteAll(s.Store, "deleted", t.Name, writes) // all or none
	if err != nil {
		return fail("Nothing was deleted, as %s %s could not be: %v", t.Name, ids[i], err)
	}
	c, titles := oneChange("deleted", t, done)
	return toolResult{text: fmt.Sprintf("deleted %s, as one change (one Undo puts them back): %s.", schema.Count(len(ids), t.Name), schema.Names(titles)), change: &c}
}
