package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"strings"
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
		{Name: "get_record", Description: "One record of a content type, with every field, by id.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type": map[string]any{"type": "string", "enum": s.App.Types.Names(), "description": "A content type from describe."},
					"id":   map[string]any{"type": "string", "description": "The record's id, from find_records or a page URL /t/<type>/<id>."},
				},
				"required":             []string{"type", "id"},
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
	case "get_record":
		var a struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &a); err != nil {
				return "arguments were not valid JSON: " + err.Error(), true
			}
		}
		if a.Type == "" || a.ID == "" {
			return "get_record needs type and id", true
		}
		rec, err := s.App.Store.Get(strings.ToLower(a.Type), a.ID)
		if err != nil {
			return fmt.Sprintf("no %s with id %s: %v. Use find_records to get an id", a.Type, a.ID, err), true
		}
		raw, err := json.MarshalIndent(rec, "", "  ")
		if err != nil {
			return err.Error(), true
		}
		return string(raw), false
	}
	return s.App.Chat.Call(name, args)
}
