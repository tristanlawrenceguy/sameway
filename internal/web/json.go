package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// APIError is the JSON error shape. Code is stable, Message is for people,
// Fields lists per-field validation problems when there are any.
type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// WriteJSON answers with v as indented JSON.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// WriteError answers with an error in the JSON error shape: invalid
// fields, not found, or a bad request.
func WriteError(w http.ResponseWriter, err error) {
	var ve *schema.ValidationError
	switch {
	case errors.As(err, &ve):
		WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": APIError{Code: "invalid", Message: "some fields are invalid", Fields: ve.Problems}})
	case errors.Is(err, store.ErrNotFound):
		WriteJSON(w, http.StatusNotFound, map[string]any{"error": APIError{Code: "not_found", Message: err.Error()}})
	default:
		WriteJSON(w, http.StatusBadRequest, map[string]any{"error": APIError{Code: "bad_request", Message: err.Error()}})
	}
}
