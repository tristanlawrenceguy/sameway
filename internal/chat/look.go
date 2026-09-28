package chat

import (
	"context"
	"encoding/json"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The assistant sees a page the way the person gets it. Without this it
// knows only what its own tools said, so "this button does nothing" is a
// guess, and so is "done" after it builds something. With it, it reads
// the page with its scripts run, does what the person did there, and
// sees what they see: what is on it, what is hidden, where Tab goes,
// what went wrong. It changes nothing.

// lookLimit is as much of a reading as goes back to the model; a longer
// one says how to narrow it.
const lookLimit = 16000

var lookTool = llm.Tool{Name: "look_at_page",
	Description: "See a page of this workspace the way the person gets it: its headings, landmarks and controls with what they hold, what is hidden, where Tab goes, and every script error and structural problem. Use it when the person says something does not work, look right, or cannot be reached, doing what they did as steps, and to check a page after you change it, before saying it is done. It changes nothing.",
	Schema: obj(map[string]any{
		"path": map[string]any{"type": "string", "description": "The page, such as /t/note/abc or /c/work. Defaults to the tab the person is on."},
		"steps": map[string]any{"type": "array", "description": "What the person did there first, in order. Controls and fields are found by the name a screen reader says.",
			"items": map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{
				"press": map[string]any{"type": "string", "description": "Press the control with this name."},
				"type":  map[string]any{"type": "string", "description": "Type these words, into the focused field or the one named by into."},
				"into":  map[string]any{"type": "string", "description": "The field to type into, by name."},
				"key":   map[string]any{"type": "string", "description": "Press a key: Tab, Shift+Tab, Enter, Escape, Space, an arrow key, Home, End, Backspace or Delete."},
				"wait":  map[string]any{"type": "integer", "description": "Wait this many milliseconds."},
			}}},
		"only": map[string]any{"type": "array", "items": map[string]any{"type": "string", "enum": []string{"landmarks", "headings", "controls", "live", "components"}}, "description": "Keep only these parts of a long page; problems always stay."},
		"kind": map[string]any{"type": "string", "description": "Keep only controls of this kind: link, button, textbox, checkbox, radio, listbox, disclosure."},
		"name": map[string]any{"type": "string", "description": "Keep only controls with these words in their name."},
	})}

// lookTools is the look, when the server has lent the way to take it.
func (s *Service) lookTools() []llm.Tool {
	if s.Look == nil {
		return nil
	}
	return []llm.Tool{lookTool}
}

// lookAtPage reads a page for the model, on the tab the person is on
// when none is named.
func (s *Service) lookAtPage(raw json.RawMessage) toolResult {
	if s.Look == nil {
		return fail("looking at pages is not available here")
	}
	var ask map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &ask); err != nil {
			return fail("arguments were not valid JSON: %v", err)
		}
	}
	if ask == nil {
		ask = map[string]any{}
	}
	if p, _ := ask["path"].(string); p == "" {
		ask["path"] = "/"
		if s.current != "" {
			ask["path"] = "/c/" + s.current
		}
	}
	out, err := s.Look(context.Background(), ask)
	if err != nil {
		return fail("%v", err)
	}
	if len(out) > lookLimit {
		out = out[:lookLimit] + "… (cut short: ask again with only, kind or name for the part you need)"
	}
	return toolResult{text: out}
}
