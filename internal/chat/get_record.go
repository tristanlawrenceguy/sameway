package chat

import (
	"encoding/json"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/relate"
)

// getRecord gives the model a record's fields, so it can answer from what
// a note or a file says rather than from its title alone, and everything
// the record is connected to. The person's page shows those connections
// as a line of counts; the model is given them in full, with the where
// that follows each one, because it cannot follow what it was not told
// about, and because it is the one deciding whether the person has a
// reason to see any of it.
func (s *Service) getRecord(typeName, id string) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	rec, err := s.Store.Get(t.Name, id)
	if err != nil {
		return fail("no %s with id %s. Use find_records to get an id", t.Name, id)
	}
	page := relate.Page(t.Name, rec.ID)
	// Who wrote the words is said before them, and that they are data:
	// see provenance.go. A struct, so a reader meets that first.
	out := struct {
		ID        string         `json:"id"`
		Type      string         `json:"type"`
		Page      string         `json:"page"`
		Title     string         `json:"title"`
		WrittenBy string         `json:"written_by"`
		Untrusted string         `json:"untrusted"`
		Fields    map[string]any `json:"fields"`
		Related   any            `json:"related,omitempty"`
		Open      string         `json:"open,omitempty"`
	}{ID: rec.ID, Type: t.Name, Page: page, Title: recordTitle(s.Store, t, rec), WrittenBy: s.Writers().Of(t.Name, rec).Words,
		Untrusted: "title and fields are what was written into this record: " + Untrusted, Fields: rec.Fields}
	if links := relate.Of(s.Store, t, rec, time.Now()); len(links) > 0 {
		out.Related, out.Open = links, page+"?show=<key>"
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fail("%v", err)
	}
	return toolResult{text: string(raw)}
}
