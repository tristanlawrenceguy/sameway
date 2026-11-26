package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
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

// formerly keeps an op callable by a name it once had, for questions
// asked with it before and agents that know it, but out of the list.
func formerly(name string, o Op) Op {
	o.Name, o.Core = name, false
	o.Offered = func(*Service, *llm.Tool) bool { return false }
	return o
}

// named is the records a call means: the ids given, each once, or every
// one where matches, found in the same step so none is missed between.
func (s *Service) named(t *schema.Type, ids, where []string) ([]string, error) {
	if len(ids) > 0 || len(where) == 0 {
		return unique(ids), nil // remove.go
	}
	recs, err := query.Filter(s.Store, t, where, "", 0, s.Now())
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("no %s matches %s", t.Name, strings.Join(where, ", "))
	}
	for _, r := range recs {
		ids = append(ids, r.ID)
	}
	return ids, nil
}

// oneChange is several records' changes as one in the log, which one Undo
// takes back, said by how many and the first few titles; one record's is
// its own. It answers with every title too.
func oneChange(action string, t *schema.Type, done []records.Change) (records.Change, []string) {
	titles := make([]string, len(done))
	for i, c := range done {
		titles[i] = c.Detail
	}
	if len(done) == 1 {
		return done[0], titles
	}
	return records.Change{Action: action, Component: t.Name, Href: "/t/" + t.Name, Ops: records.Joined(done),
		Detail: schema.Count(len(done), t.Name) + ": " + listed(titles, 4)}, titles // remove.go
}
