package mcp

import (
	"encoding/json"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// The assistant in the app whose model runs its tools here (sameway mcp
// --assistant, from llm.Command) is given what the assistant in the app
// is given: the tools its turn speaks of, named by --tools, and
// more_tools for the rest (chat/more_tools.go), not every tool every
// turn. A tool more_tools brings is listed from then on, and the client is
// told the list changed, so it reads it again with the tool's own
// annotations. Any other client is given every tool, as MCP clients
// expect: one that does not read the list again would never find the
// rest.

// turnTools is out cut to the turn's tools and more_tools, when Only says
// which; the tools of this server's own (describe, look, today, try) stay.
func (s *Server) turnTools(out []tool) []tool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Only == nil {
		return out
	}
	var kept []tool
	var all, given []llm.Tool
	for _, t := range s.App.Chat.Tools() {
		all = append(all, t)
		if s.Only[t.Name] {
			given = append(given, t)
		}
	}
	chats := map[string]bool{}
	for _, t := range all {
		chats[t.Name] = true
	}
	for _, t := range out {
		if !chats[t.Name] || s.Only[t.Name] {
			kept = append(kept, t)
		}
	}
	if more, ok := chat.MoreTool(all, given); ok {
		kept = append(kept, tool{Name: more.Name, Description: more.Description, InputSchema: more.Schema})
	}
	return kept
}

// brought lists the tool more_tools was asked for from now on, and tells
// the client its list changed.
func (s *Server) brought(args json.RawMessage) {
	var a struct{ Name string }
	json.Unmarshal(args, &a)
	s.mu.Lock()
	if s.Only == nil || s.Only[a.Name] {
		s.mu.Unlock()
		return
	}
	s.Only[a.Name] = true
	s.mu.Unlock()
	raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/tools/list_changed"})
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Out.Write(append(raw, '\n'))
}
