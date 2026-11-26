package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A brain dump of seven tasks was made with seven create_record calls,
// thirty-three seconds of round-trips, each a place a model could stop
// half way, and the receipt said seven changes with seven Undos. Several
// records of one type are one call and one change instead: all are made
// or none, and one Undo takes them all back.

func (s *Service) createRecords(typeName string, list []map[string]any) toolResult {
	for _, fields := range list {
		if _, ok := fields["type"]; ok && typeName == "" {
			return fail("nothing was made: records are all of one type, given beside them as type, not in each one. For records of several types, call once per type.")
		}
	}
	t, err := s.contentType(typeName)
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
		return fail("Nothing was made, as records[%d]%s could not be: %s Call again with all of them, that one fixed.",
			i, calledSaid(t, list[i]), fieldHelp(err, t))
	}
	c, titles := oneChange("created", t, done) // record_ops.go
	var lines []string
	for i, rec := range made {
		lines = append(lines, fmt.Sprintf("%s %q at /t/%s/%s.", rec.ID, titles[i], t.Name, rec.ID)+
			s.datesSaid(t, list[i], rec, false)) // dates_said.go; the rest is the change's notes (notice.go)
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
