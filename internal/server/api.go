package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// apiError is the JSON error shape. code is stable, message is for people,
// fields lists per-field validation problems when there are any.
type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	var ve *schema.ValidationError
	switch {
	case errors.As(err, &ve):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "invalid", Message: "some fields are invalid", Fields: ve.Problems}})
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]any{"error": apiError{Code: "not_found", Message: err.Error()}})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": apiError{Code: "bad_request", Message: err.Error()}})
	}
}

func readBody(r *http.Request) (map[string]any, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var fields map[string]any
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, errors.New("body must be a JSON object of fields: " + err.Error())
	}
	return fields, nil
}

func (s *Server) apiDescribe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Describe())
}

func (s *Server) apiList(w http.ResponseWriter, r *http.Request) {
	recs, err := s.app.Store.List(r.PathValue("type"), store.ListOptions{OrderBy: r.URL.Query().Get("order"), Desc: r.URL.Query().Get("dir") == "desc"})
	if err != nil {
		writeError(w, err)
		return
	}
	if recs == nil {
		recs = []*store.Record{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"type": r.PathValue("type"), "count": len(recs), "records": recs})
}

func (s *Server) apiGet(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(r.PathValue("type"), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) apiCreate(w http.ResponseWriter, r *http.Request) {
	fields, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rec, err := s.app.Store.Create(r.PathValue("type"), fields)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Location", "/api/"+rec.Type+"/"+rec.ID)
	writeJSON(w, http.StatusCreated, rec)
}

func (s *Server) apiUpdate(w http.ResponseWriter, r *http.Request) {
	fields, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	rec, err := s.app.Store.Update(r.PathValue("type"), r.PathValue("id"), fields)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) apiDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Store.Delete(r.PathValue("type"), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": r.PathValue("id")})
}

// apiChat lets an agent talk to the assistant the same way a person does.
func (s *Server) apiChat(w http.ResponseWriter, r *http.Request) {
	body, err := readBody(r)
	if err != nil {
		writeError(w, err)
		return
	}
	text, _ := body["message"].(string)
	rec, err := s.app.Chat.Send(r.Context(), text)
	if rec == nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reply": rec, "ok": err == nil})
}
