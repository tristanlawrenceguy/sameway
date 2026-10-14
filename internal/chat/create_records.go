package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A brain dump of seven tasks was made with seven create_record calls,
// thirty-three seconds of round-trips, each a place a model could stop
// half way, and the receipt said seven changes with seven Undos. Several
// records of one type are one call and one change instead: all are made
// or none, and one Undo takes them all back.

var createRecordsOp = Op{Title: "Make several records",
	Words: []string{"list", "these", "several", "all of", "each", "brain dump"},
	Doing: func(a callArgs) string {
		if len(a.Records) == 0 {
			return "Adding " + or(schema.Plural(a.Type), "records")
		}
		return "Adding " + schema.Count(len(a.Records), a.Type)
	},
	Run: func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.createRecords(a.Type, a.Records) },
	Tool: llm.Tool{Name: "create_records", Description: "Make several records of one content type in one change: a list of tasks, the people in a message, each thing the person named. All are made or none: if one is wrong, nothing is made and you are told which. One Undo takes them all back. Each entry of records is what create_record takes as fields. Returns each new record's id and page.",
		Schema: obj(map[string]any{
			"type":    typeArg,
			"records": map[string]any{"type": "array", "description": "One object of field values per record, each matching the type's schema, as create_record's fields.", "items": map[string]any{"type": "object"}},
		}, "type", "records")}, Offered: typed}

func (s *Service) createRecords(typeName string, list []map[string]any) toolResult {
	t, err := records.ContentType(s.Store, typeName)
	if err != nil {
		return fail("%v", err)
	}
	if len(list) == 0 {
		return fail("records is empty: pass one object of fields per record to make, as create_record's fields")
	}
	writes := make([]records.Writing, len(list))
	for i, fields := range list {
		if fields == nil {
			list[i] = map[string]any{}
		}
		writes[i] = records.Writing{Fields: list[i]}
	}
	made, done, i, err := records.WriteAll(s.Store, "created", t.Name, writes) // all or none
	if err != nil {
		return fail("Nothing was made: records[%d]%s could not be saved — %s. %s Fix that one and call create_records again with all of them.",
			i, calledSaid(t, list[i]), humanizeValidationError(err.Error()), fieldHelp(err, t))
	}
	ops, titles := records.Joined(done), make([]string, len(done))
	for i, c := range done {
		titles[i] = c.Detail
	}
	var lines []string
	for i, rec := range made {
		lines = append(lines, fmt.Sprintf("%s %q at /t/%s/%s.", rec.ID, titles[i], t.Name, rec.ID)+
			s.datesSaid(t, list[i], rec, false)) // dates_said.go; the rest is the change's notes (notice.go)
	}
	c := records.Change{Action: "created", Component: t.Name, Href: "/t/" + t.Name, Ops: ops,
		Detail: schema.Count(len(made), t.Name) + ": " + listed(titles, 4)} // remove.go: the log says the first few, as for blocks
	if len(made) == 1 { // one is said as create_record says it, by its page
		c.ID, c.Detail, c.Href = made[0].ID, titles[0], "/t/"+t.Name+"/"+made[0].ID
	}
	return toolResult{text: fmt.Sprintf("created %s, as one change (one Undo takes them all back):\n%s", schema.Count(len(made), t.Name), strings.Join(lines, "\n")), change: &c}
}

// calledSaid is the title a refused entry was given, as ` ("Book the
// car")`, so the model can find it in its list; "" when it has none.
func calledSaid(t *schema.Type, fields map[string]any) string {
	title := strings.TrimSpace(t.Called("", fields))
	if title == schema.Words(t.Name) {
		return ""
	}
	return fmt.Sprintf(" (%q)", title)
}
