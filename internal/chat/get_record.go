package chat

import (
	"encoding/json"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// getRecord gives the model a record's fields, so it can answer from what
// a note or a file says rather than from its title alone, and everything
// the record is connected to. The person's page shows those connections
// as a line of counts; the model is given them in full, with the where
// that follows each one, because it cannot follow what it was not told
// about, and because it is the one deciding whether the person has a
// reason to see any of it.
func (s *Service) getRecord(typeName, id string) toolResult {
	t, err := records.ContentType(s.Store, typeName)
	if err != nil {
		return fail("%v", err)
	}
	rec, err := s.Store.Get(t.Name, id)
	if err != nil {
		return fail("no %s with id %s. Use find_records to get an id", t.Name, id)
	}
	out := s.RecordView(t, rec) // view.go: the same as the API gives
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fail("%v", err)
	}
	return toolResult{text: string(raw)}
}
