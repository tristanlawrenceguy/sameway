package server

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Deleting a record. One step, because it can be taken back: what it was
// goes into the activity log, and the listing the person lands on offers
// to put it back. No page asks "are you sure".

func (s *Server) deleteForm(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	list := "/t/" + t.Name
	rec, err := s.app.Store.Get(t.Name, r.PathValue("id"))
	if err != nil {
		s.failed(w, r, "Not deleted", err, list)
		return
	}
	title := s.title(t, rec)
	// Logged with what it was, so the deletion can be undone, from the
	// message that says it happened.
	_, undo, err := records.WriteAs(s.app.Store, s.who(r), "deleted", t.Name, rec.ID, nil)
	if err != nil {
		s.failed(w, r, "Not deleted", err, list)
		return
	}
	// Back where the person was, unless that was the record's own page,
	// which is gone: then its list.
	back := web.BackOf(r, list)
	if own := list + "/" + rec.ID; back == own || strings.HasPrefix(back, own+"/") || strings.HasPrefix(back, own+"?") {
		back = list
	}
	s.tellAt(w, r, outcome{Title: "Deleted", Text: title + " is deleted.", Undo: undo}, back)
}
