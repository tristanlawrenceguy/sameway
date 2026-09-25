package server

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Writes through the API are changes like any other: logged as the
// person's, through the API, with what each record was before, so each
// can be undone. The activity log itself is not written here at all.

func (s *Server) apiCreate(w http.ResponseWriter, r *http.Request) {
	if s.keptLog(w, r) {
		return
	}
	fields, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if s.keptFromVisitor(w, r, fields) {
		return
	}
	rec, err := s.app.Store.Create(r.PathValue("type"), fields)
	if err != nil {
		writeError(w, err)
		return
	}
	chat.RecordWrite(s.app.Store, chat.ThroughAPI, "created", rec, nil)
	w.Header().Set("Location", "/api/"+rec.Type+"/"+rec.ID)
	writeJSON(w, http.StatusCreated, rec)
}

func (s *Server) apiUpdate(w http.ResponseWriter, r *http.Request) {
	if s.keptLog(w, r) {
		return
	}
	fields, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if s.keptFromVisitor(w, r, fields) {
		return
	}
	was, err := s.app.Store.Get(r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	rec, err := s.app.Store.Update(r.PathValue("type"), r.PathValue("id"), fields)
	if err != nil {
		writeError(w, err)
		return
	}
	chat.RecordWrite(s.app.Store, chat.ThroughAPI, "updated", rec, was.Fields)
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) apiDelete(w http.ResponseWriter, r *http.Request) {
	if s.keptLog(w, r) {
		return
	}
	was, err := s.app.Store.Get(r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.app.Store.Delete(r.PathValue("type"), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	// Logged with everything it had, so it can be put back.
	chat.RecordWrite(s.app.Store, chat.ThroughAPI, "deleted", was, was.Fields)
	writeJSON(w, http.StatusOK, map[string]any{"deleted": r.PathValue("id")})
}

// keptLog refuses a write to the activity log itself: it is what makes
// every other change reversible, so it is kept by Sameway, read by anyone,
// and changed by nobody. A change is taken back by undoing it.
func (s *Server) keptLog(w http.ResponseWriter, r *http.Request) bool {
	if r.PathValue("type") != chat.ActivityType {
		return false
	}
	writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": apiError{Code: "kept",
		Message: "the activity log is kept by Sameway and cannot be changed; to take a change back, POST /activity/<id>/undo"}})
	return true
}
