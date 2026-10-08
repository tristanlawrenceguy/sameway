package server

import (
	"github.com/tristanlawrenceguy/sameway/internal/blocks"
)

// blockCheck resolves a block's records the way its page will, without
// drawing it: what it would show, in a few words, or why it cannot be
// shown, in the very words the page would say it, since it is the same
// resolving. Every write of a block goes through it (chat.Service.Check),
// so a block that could only say it is set up wrong is refused when it is
// written, not found broken after the one who wrote it has said "done".
func (s *Server) blockCheck(component string, props map[string]any) (shows, problem string) {
	k, ok := blocks.Of(component)
	if !ok || k.Shows == nil {
		return "", ""
	}
	out := s.resolve(component, props, "", nil)
	if p, _ := out["problem"].(string); p != "" {
		return "", p
	}
	if p := s.meaningProblem(component, props); p != "" {
		return "", p
	}
	return k.Shows(s.app.Blocks, props, out), ""
}

// noType says a type is not there, and what is (blocks.NoType).
func (s *Server) noType(name string) string { return s.app.Blocks.NoType(name) }
