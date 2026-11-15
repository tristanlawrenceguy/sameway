package mcp

import (
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
)

// A model that runs its tools itself, such as Claude Code, reaches the
// workspace through this server rather than through the assistant's own
// loop. When the turn works for someone other than the owner (For), this
// server has their permissions and nothing more: it offers exactly the
// tools the assistant has for them (chat.Service.Tools, of which the
// turn was given Only), runs each through their service, where a change
// beyond them becomes a request and a kind they cannot reach is not
// there, and offers none of its own tools (look and today read as the
// owner). more_tools brings only what is theirs.

// forTools are the tools offered when the server works for someone.
func (s *Server) forTools(svc *chat.Service) []tool {
	theirs := svc.Tools()
	s.mu.Lock()
	var given []llm.Tool
	for _, t := range theirs {
		if s.Only == nil || s.Only[t.Name] {
			given = append(given, t)
		}
	}
	s.mu.Unlock()
	var out []tool
	for _, t := range given {
		out = append(out, tool{Name: t.Name, Description: t.Description, InputSchema: t.Schema, Annotations: annotations(t.Name)})
	}
	if more, ok := chat.MoreTool(theirs, given); ok {
		out = append(out, tool{Name: more.Name, Description: more.Description, InputSchema: more.Schema})
	}
	return out
}
