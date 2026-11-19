package chat

import (
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Making and changing records, one or several in the one tool: a model
// given a tool for one and another for several reached for the one, call
// after call, and each description spent words sending it to the other.

var createRecordOp = Op{Title: "Make a record",
	Core: true,
	Doing: func(a callArgs) string {
		if len(a.Records) > 0 {
			return "Adding " + schema.Count(len(a.Records), a.Type)
		}
		return "Adding" + or(an(schema.Words(a.Type)), " a record") + a.called()
	},
	Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
		if len(a.Records) > 0 {
			return s.createRecords(a.Type, a.Records) // create_records.go
		}
		return s.createRecord(a.Type, a.Fields)
	},
	Tool: llm.Tool{Name: "create_record", Description: "Make a record of a content type (a note, a task, any the workspace has), its fields as the type's in the catalogue; or several of one type with records, all made or none. Returns each one's page.",
		Schema: obj(map[string]any{
			"type":    typeArg,
			"fields":  map[string]any{"type": "object", "description": "Field values; one left out takes its default."},
			"records": map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "In place of fields, to make several: one object of field values each."},
		}, "type")}, Offered: makeTyped}

var updateRecordOp = Op{Title: "Change a record", Traits: Traits{Idempotent: true},
	Core: true,
	Doing: func(a callArgs) string {
		if len(a.IDs) > 0 {
			return "Updating " + schema.Count(len(a.IDs), a.Type)
		}
		return "Updating" + or(an(schema.Words(a.Type)), " a record")
	},
	Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult {
		if a.ID == "" && (len(a.IDs) > 0 || len(a.Where) > 0) {
			return s.updateRecords(a.Type, a.IDs, a.Where, a.Fields) // update_records.go
		}
		return s.updateRecord(a.Type, a.ID, a.Fields, a.Version)
	},
	Tool: llm.Tool{Name: "update_record", Description: "Change fields on records that exist; only the fields given change. Name one by id, several by ids, or every one where matches (no need to find them first): all changed or none.",
		Schema: obj(map[string]any{
			"type":    typeArg,
			"id":      map[string]any{"type": "string", "description": "From find_records, or a page's /t/<type>/<id>."},
			"ids":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"where":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "As find_records takes it."},
			"fields":  map[string]any{"type": "object", "description": "What to change, to what."},
			"version": map[string]any{"type": "string", "description": "get_record's version: if it changed since, nothing is written and you see it as it is now."},
		}, "type", "fields")}, Offered: typed}
