package api

import (
	"errors"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// A type or a record that is not there is 404, said in the words the
// assistant's tools and the CLI use (records/refused.go), with where to
// look next. A type that was not there was 400 "no content type", and a
// record 404 "not found", which said neither which nor what to do.

// knownType refuses a path whose type the workspace does not have, and
// says whether the request may go on.
func (s *Service) knownType(w http.ResponseWriter, r *http.Request) bool {
	typ := r.PathValue("type")
	if _, ok := s.app.Types.Get(typ); ok {
		return true
	}
	web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": web.APIError{Code: "not_found",
		Message: records.NoType(typ, s.app.Store.Types().Names()) + "; GET /api/describe/types says what each takes"}})
	return false
}

// writeErr answers err as web.WriteError does, but a record that is not
// there is named, with the list that has those there are.
func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	if !errors.Is(err, store.ErrNotFound) {
		web.WriteError(w, err)
		return
	}
	typ, id := r.PathValue("type"), r.PathValue("id")
	web.WriteJSON(w, http.StatusNotFound, map[string]any{"error": web.APIError{Code: "not_found",
		Message: records.NoRecord(typ, id) + "; GET /api/" + typ + " lists those there are"}})
}
