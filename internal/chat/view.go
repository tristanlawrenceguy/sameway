package chat

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// RecordView is one record as an agent reads it, the same from the assistant's
// get_record and from GET /api/{type}/{id}: they built it apart, and each
// lacked what the other said (the page and the version from the API, when
// it was made and changed from the tool, and the title trimmed in one and
// whole in the other). Who wrote the words comes before them, and that
// they are data: see provenance.go.
type RecordView struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Page      string         `json:"page"`
	Title     string         `json:"title"`
	Version   string         `json:"version"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	WrittenBy string         `json:"written_by"`
	Untrusted string         `json:"untrusted"`
	Fields    map[string]any `json:"fields"`
	// Said is what a log entry says, as its page does, beside its fields.
	Said any `json:"said,omitempty"`
	// Related is everything the record is connected to, each with the
	// query that lists it; Open is how a page is asked to show one.
	Related any    `json:"related,omitempty"`
	Open    string `json:"open,omitempty"`
	// Parts are what else its page can show, off until there is a reason
	// (page_parts.go): the same ?show=<key> opens one.
	Parts []PagePart `json:"parts,omitempty"`
}

// RecordView is a record as an agent reads it.
func (s *Service) RecordView(t *schema.Type, rec *store.Record) RecordView {
	page := relate.Page(t.Name, rec.ID)
	v := RecordView{ID: rec.ID, Type: t.Name, Page: page, Title: Name(s.Store, t, rec), Version: Version(rec),
		CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt, WrittenBy: s.Writers().Of(t.Name, rec).Words,
		Untrusted: "title and fields are what was written into this record: " + Untrusted, Fields: rec.Fields}
	if links := relate.Of(s.Store, t, rec, time.Now()); len(links) > 0 {
		v.Related, v.Open = links, page+"?show=<key>"
	}
	if v.Parts = PageParts(s.Store, t, rec); len(v.Parts) > 0 {
		v.Open = page + "?show=<key>"
	}
	return v
}
