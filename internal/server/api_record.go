package server

import (
	"fmt"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// One record over the API, and everything it is connected to.
//
// The record's own page shows those connections as a line of counts,
// because a page at rest says nothing. Here they are in full, each with
// the query that lists it, because an agent cannot follow a connection it
// was never told about and has no page to run out of room on. Same
// question, same answer, different amount of it shown: see related.go and
// internal/relate.

func (s *Server) apiGet(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	only, err := s.onlyFields(r, rec.Type)
	if err != nil {
		writeError(w, err)
		return
	}
	t, ok := s.app.Types.Get(rec.Type)
	if !ok {
		writeError(w, fmt.Errorf("no content type %q", rec.Type))
		return
	}
	// One view of a record, the assistant's get_record's too (chat/view.go).
	out := s.app.Chat.RecordView(t, rec)
	out.Fields = trimmed(rec, only).Fields
	// An entry in the log keeps its fields as they are, data to act on, and
	// says itself as its page does (activity_page.go).
	if rec.Type == chat.ActivityType {
		out.Said = s.activityFacts(rec)
		// Resolve raw type identifiers in the detail field so agents read
		// display names, not schema keys. This mirrors what say() does for
		// the HTML event component (lines.go:59-61).
		if target, _ := rec.Fields["target"].(string); target == "type" {
			if detail, ok := out.Fields["detail"].(string); ok && detail != "" {
				out.Fields["detail"] = schema.DisplayName(detail)
			}
		}
	}
	// The version to send back with If-Match, so a change made from it is
	// refused when the record has moved on (versions.go).
	w.Header().Set("ETag", `"`+chat.Version(rec)+`"`)
	writeJSON(w, http.StatusOK, out)
}

// A record over the API says its title, the one its page is headed with
// and its row in a list shows: a note's title, an entry's habit and how
// much ("Read: 25 minutes"). An agent should not have to read a page, or
// work out a habit's unit, to know what a record is called.
func (s *Server) apiTitle(rec *store.Record) string {
	if t, ok := s.app.Types.Get(rec.Type); ok {
		return s.title(t, rec)
	}
	return rec.ID
}

// titled is a record as it always was over the API, with its title
// beside it: what a write answers with.
type titled struct {
	*store.Record
	Title string `json:"title"`
}

func (s *Server) titled(rec *store.Record) titled {
	return titled{rec, s.apiTitle(rec)}
}
