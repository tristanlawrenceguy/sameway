package mcp

import (
	"context"
	"encoding/json"
	"sort"
)

// A connection from the internet reads what the owner published, and only
// that: describe tells it the published content types, and find_records
// and get_record answer for those types alone. Published, set by the
// command line, says which types those are just now.

// publicCall answers a call from the internet, or says it may not.
func (s *Server) publicCall(ctx context.Context, name string, args json.RawMessage) (string, bool, bool) {
	if !s.reachOf(ctx).public {
		return "", false, false
	}
	types := map[string]bool{}
	if s.Published != nil {
		types = s.Published()
	}
	if name == "describe" {
		var out []any
		var names []string
		for t := range types {
			names = append(names, t)
		}
		sort.Strings(names)
		for _, n := range names {
			if t, ok := s.App.Types.Get(n); ok {
				out = append(out, map[string]any{"name": t.Name, "description": t.Description, "fields": t.JSONSchema()})
			}
		}
		raw, _ := json.MarshalIndent(map[string]any{"published": out, "tools": "find_records and get_record, for these types"}, "", "  ")
		return string(raw), false, true
	}
	var a struct {
		Type string `json:"type"`
	}
	json.Unmarshal(args, &a)
	if !types[a.Type] {
		return "only published content can be read here: " + joined(types), true, true
	}
	text, isErr := s.App.Chat.Call(name, args)
	return text, isErr, true
}

func joined(types map[string]bool) string {
	var names []string
	for t := range types {
		names = append(names, t)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "nothing is published"
	}
	out := names[0]
	for _, n := range names[1:] {
		out += ", " + n
	}
	return out
}
