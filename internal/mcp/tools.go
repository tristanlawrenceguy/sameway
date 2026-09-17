package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"net/http"
	"net/http/httptest"
)

// tool is what MCP calls a tool: the same shape the chat service already
// holds, under the names the protocol expects.
type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// tools lists what a client may call: reading, which the assistant does
// through its prompt and an agent cannot, and then the assistant's own list.
func (s *Server) tools() []tool {
	out := []tool{
		{Name: "describe", Description: "Everything about this workspace: content types with their field schemas, components with their manifests, the assistant's tools, and every route. Call it first. Give part to read one section, and name for one item in it: part types, name note is the fields of a note.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"part": map[string]any{"type": "string", "enum": app.Parts, "description": "One section of the description, or omit for all of it."},
					"name": map[string]any{"type": "string", "description": "One item in that section: a type, component or tool name, or a route key."},
				},
				"additionalProperties": false,
			}},
		{Name: "look", Description: "A page as a screen reader gets it, without a browser: title, landmarks, headings, controls with where they lead, live regions, the components on it, and its structural problems. Give path for a page; method and form to do what a person does and read where they land; or component and props to read one component rendered from props.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":      map[string]any{"type": "string", "description": "A page on this server, such as /t/note or /activity."},
					"method":    map[string]any{"type": "string", "enum": []string{"GET", "POST"}, "description": "POST to submit form as a person would; defaults to GET, or POST when form is given."},
					"form":      map[string]any{"type": "object", "description": "Form fields to submit, by name."},
					"component": map[string]any{"type": "string", "description": "A component from describe, to read on its own instead of a page."},
					"props":     map[string]any{"type": "object", "description": "Props for that component."},
				},
				"additionalProperties": false,
			}},
	}
	for _, t := range s.App.Chat.Tools() {
		out = append(out, tool{Name: t.Name, Description: t.Description, InputSchema: t.Schema})
	}
	return out
}

// call runs one tool and returns what the client is told. The reading tools
// live here; everything that changes something goes through the chat service,
// so an agent gets the same checks and the same activity log as the assistant.
func (s *Server) call(ctx context.Context, name string, args json.RawMessage) (string, bool) {
	switch name {
	case "describe":
		var a struct {
			Part string `json:"part"`
			Name string `json:"name"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &a); err != nil {
				return "arguments were not valid JSON: " + err.Error(), true
			}
		}
		v, err := s.App.Describe().Part(a.Part, a.Name)
		if err != nil {
			return err.Error(), true
		}
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err.Error(), true
		}
		return string(raw), false
	case "look":
		// The same handler the HTTP API has, so a page reads the same either way.
		req := httptest.NewRequest(http.MethodPost, "/api/look", bytes.NewReader(args))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		s.web().ServeHTTP(rec, req)
		return rec.Body.String(), rec.Code >= 400
	}
	// get_record, find_records and the rest are the assistant's own tools.
	return s.App.Chat.Call(name, args)
}

// web is the HTTP server over the same app, for the tools that read a page
// the way the API does. Built once, on first use.
func (s *Server) web() http.Handler {
	if s.http == nil {
		s.http = server.New(s.App)
	}
	return s.http
}
