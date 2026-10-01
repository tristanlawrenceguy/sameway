package mcp

import (
	"context"
	"encoding/json"
)

// try runs a tool on a throwaway copy of the workspace (app.Sandbox) and
// answers what it answered, so an agent finds out whether a change would
// be refused, and how, without making it: the same checks, the same
// words, because it is the same code on the same records. What reaches
// beyond the workspace is not tried, since a copy cannot hold it back.

var tryTool = tool{Name: "try", Description: "Try a change without making it: name a tool and its arguments, and it runs on a throwaway copy of the workspace and answers what that tool would. A refusal is the refusal you would get; a success is what would have happened, and nothing has. Ids it gives belong to the copy. Not for run_action or update_sameway, which reach outside the workspace.",
	InputSchema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":      map[string]any{"type": "string", "description": "The tool to try, such as create_record or add_component."},
			"arguments": map[string]any{"type": "object", "description": "Its arguments, as you would send them to it."},
		},
		"required":             []string{"name", "arguments"},
		"additionalProperties": false,
	}}

func (s *Server) try(ctx context.Context, args json.RawMessage) (string, bool) {
	var a struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(args, &a); err != nil || a.Name == "" {
		return "try needs name, the tool to try, and arguments, its arguments", true
	}
	tr, known := toolTraits[a.Name]
	switch {
	case !known || a.Name == "try":
		return "there is no tool " + a.Name + " to try; tools/list has them", true
	case tr.openWorld:
		return a.Name + " reaches outside the workspace, so a copy cannot try it; ask the person before you run it", true
	case tr.readOnly:
		return a.Name + " changes nothing, so call it as it is", true
	}
	if ok, _ := s.may(ctx, a.Name); !ok {
		return refusedOff, true
	}
	sb, done, err := s.App.Sandbox()
	if err != nil {
		return "the workspace could not be copied to try it: " + err.Error(), true
	}
	defer done()
	there := &Server{App: sb, Version: s.Version, Assistant: s.Assistant}
	_, svc := there.may(ctx, a.Name)
	text, isError := there.run(ctx, there.forAgent(ctx, svc), a.Name, a.Arguments)
	if isError {
		return "Tried on a copy; nothing was changed. " + a.Name + " would be refused: " + text, true
	}
	return "Tried on a copy; nothing was changed. " + a.Name + " would answer: " + text, false
}
