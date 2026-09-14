package server

import (
	"net/http"
)

// deleteForm handles POST /t/{type}/{id}/delete, removing the record and redirecting.
func (s *Server) deleteForm(w http.ResponseWriter, r *http.Request) {
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	id := r.PathValue("id")
	err := s.app.Store.Delete(t.Name, id)
	if err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, "/t/"+t.Name, http.StatusSeeOther)
}
