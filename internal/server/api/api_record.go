package api

import (
	"fmt"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// One record over the API, and everything it is connected to.
//
// The record's own page shows those connections as a line of counts,
// because a page at rest says nothing. Here they are in full, each with
// the query that lists it, because an agent cannot follow a connection it
// was never told about and has no page to run out of room on. Same
// question, same answer, different amount of it shown: see related.go and
// internal/relate.

func (s *Service) apiGet(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		web.WriteError(w, err)
		return
	}
	only, err := s.onlyFields(r, rec.Type)
	if err != nil {
		web.WriteError(w, err)
		return
	}
	t, ok := s.app.Types.Get(rec.Type)
	if !ok {
		web.WriteError(w, fmt.Errorf("no content type %q", rec.Type))
		return
	}
	// One view of a record, the assistant's get_record's too (records/view.go).
	out := s.app.Records.ViewOf(t, rec, s.app.Records.WritersFor(false))
	out.Fields = trimmed(rec, only).Fields
	// An entry in the log keeps its fields as they are, data to act on, and
	// says itself as its page does (server/activity_page.go).
	if rec.Type == records.ActivityType {
		out.Said = s.ActivityFacts(rec)
		// Resolve raw type identifiers in the detail field so agents read
		// display names, not schema keys. This mirrors what say() does for
		// the HTML event component (server/lines.go).
		if target, _ := rec.Fields["target"].(string); target == "type" {
			if detail, ok := out.Fields["detail"].(string); ok && detail != "" {
				out.Fields["detail"] = schema.DisplayName(detail)
			}
		}
	}
	// The version to send back with If-Match, so a change made from it is
	// refused when the record has moved on (records/versions.go).
	w.Header().Set("ETag", `"`+records.Version(rec)+`"`)
	web.WriteJSON(w, http.StatusOK, out)
}

// A record over the API says its title, the one its page is headed with
// and its row in a list shows: a note's title, an entry's habit and how
// much ("Read: 25 minutes"). An agent should not have to read a page, or
// work out a habit's unit, to know what a record is called.
func (s *Service) apiTitle(rec *store.Record) string {
	if t, ok := s.app.Types.Get(rec.Type); ok {
		return s.Title(t, rec)
	}
	return rec.ID
}

// titled is a record as it always was over the API, with its title
// beside it: what a write answers with.
type titled struct {
	*store.Record
	Title string `json:"title"`
}

func (s *Service) titled(rec *store.Record) titled {
	return titled{rec, s.apiTitle(rec)}
}
