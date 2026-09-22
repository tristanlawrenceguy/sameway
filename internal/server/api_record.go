package server

import (
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/relate"
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
	out := map[string]any{"id": rec.ID, "type": rec.Type, "created_at": rec.CreatedAt, "updated_at": rec.UpdatedAt, "fields": rec.Fields}
	if t, ok := s.app.Types.Get(rec.Type); ok {
		if links := relate.Of(s.app.Store, t, rec, time.Now()); len(links) > 0 {
			out["related"] = links
		}
	}
	writeJSON(w, http.StatusOK, out)
}
