package server

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Ringing belongs to the server, not to a page. A reminder rings when its
// time comes whether or not anyone has the app in front of them: every
// few seconds, for as long as the server runs, what is due is marked
// rung, logged, and told beyond the page through notify, which the
// command line provides: the machine's own notification, and a command
// that can reach a phone or an inbox. An open page hears about it over
// /events (sync.go) and rings too. The ring is one of the server's
// background jobs (jobs.go).

// OnRing sets what a ring is told to, without starting the loop. A turn
// that ends with no page to hear it is told the same way; see tellDone.
func (s *Server) OnRing(notify func(title, text, url string)) { s.notify = notify }

// Ring marks every reminder whose time has come as rung, once, and tells
// the world. It says which rang.
func (s *Server) Ring(now time.Time) []*store.Record {
	rang := s.ring(now)
	if s.notify == nil {
		return rang
	}
	t, _ := s.app.Types.Get(ReminderType)
	for _, rec := range rang {
		text, url := blocks.RingWords(s.app.Store, rec)
		go s.notify(s.title(t, rec), text, s.linkTo(url))
	}
	return rang
}

// linkTo is a page of this workspace as a whole address, when the
// address this server listens on is known; the path alone otherwise.
func (s *Server) linkTo(path string) string {
	for _, k := range s.machine().KnownWorkspaces() {
		if sameDir(k.Dir, s.app.Workspace.Dir) && k.Addr != "" {
			return "http://" + k.Addr + path
		}
	}
	return path
}
