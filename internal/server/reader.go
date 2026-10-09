package server

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/search"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A page is said to the person reading it: the time it is for them, from
// the app's clock (app.Options.Clock, which a test fixes), and their clock
// face, 12 or 24 hours (workspace Hours24). Both are this server's app's,
// never the program's, so two workspaces served at once, or two tests,
// each keep their own.

// now is the time it is for the person.
func (s *Server) now() time.Time { return s.app.Records.Now() }

// h24 is whether the person reads times on the 24-hour clock.
func (s *Server) h24() bool { return s.app.Workspace.Hours24() }

// display is a stored value as the text a form or page shows, said to
// the person (blocks.Display).
func (s *Server) display(f schema.Field, v any) string { return blocks.Display(f, v, s.h24()) }

// reader is who a search is for (search.Reader).
func (s *Server) reader() search.Reader { return search.Reader{Now: s.now(), H24: s.h24()} }

// recordWays, momentWords and secondWords tell records with one name
// apart, as a block's list does (blocks/apart.go).
func (s *Server) recordWays(t *schema.Type, rec *store.Record) []string {
	return s.app.Blocks.RecordWays(t, rec)
}
func (s *Server) momentWords(at time.Time) string { return s.app.Blocks.MomentWords(at) }
func (s *Server) secondWords(at time.Time) string { return s.app.Blocks.SecondWords(at) }

// machine is where this computer keeps what is no one workspace's: the
// known list, copies, deleted workspaces (workspace.Machine).
func (s *Server) machine() workspace.Machine { return s.app.Workspace.Machine }

// keys is the file a pasted key is kept in (workspace.Machine).
func (s *Server) keys() llm.Keys { return llm.Keys(s.app.Workspace.Machine.Keys) }
