package server

import (
	"encoding/json"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// apiAddField adds a field to a content type: POST /api/types/{type}/fields
// with {name, type, description, values, to, required, default}. The same
// change the assistant makes with add_field, for an agent that speaks JSON.
func (s *Server) apiAddField(w http.ResponseWriter, r *http.Request) {
	var f schema.Field
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&f); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "send a field as JSON: {name, type, description, values, to, required, default}"})
		return
	}
	t, err := s.app.AddField(r.PathValue("type"), f)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": err.Error()})
		return
	}
	chat.Record(s.app.Store, "system", chat.Change{Action: "added", Component: "field", Detail: f.Name + " on " + t.Name, Href: "/t/" + t.Name})
	writeJSON(w, http.StatusCreated, t)
}

// apiAddType makes a content type: POST /api/types with {name,
// description, title, fields: [{name, type, ...}]}, fields in the order
// they should show.
func (s *Server) apiAddType(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Title       string         `json:"title"`
		Fields      []schema.Field `json:"fields"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "send a type as JSON: {name, description, title, fields: [{name, type, ...}]}"})
		return
	}
	t, err := s.app.AddType(&schema.Type{Name: in.Name, Description: in.Description, Title: in.Title, Fields: in.Fields})
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": err.Error()})
		return
	}
	chat.Record(s.app.Store, "system", chat.Change{Action: "added", Component: "type", Detail: t.Name, Href: "/t/" + t.Name})
	writeJSON(w, http.StatusCreated, t)
}
