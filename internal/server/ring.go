package server

import (
	"context"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Ringing belongs to the server, not to a page. A reminder rings when its
// time comes whether or not anyone has the app in front of them: every
// few seconds, for as long as the server runs, what is due is marked
// rung, logged, and told beyond the page through notify, which the
// command line provides: the machine's own notification, and a command
// that can reach a phone or an inbox. An open page hears about it over
// /clock/stream and rings too.

// StartRinging rings what is due for as long as ctx lasts. notify may be
// nil, in which case a ring shows only on open pages.
func (s *Server) StartRinging(ctx context.Context, notify func(title, text, url string)) {
	s.OnRing(notify)
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			s.Ring(time.Now())
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}

// OnRing sets what a ring is told to, without starting the loop.
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
		title := titleOf(t, rec)
		go s.notify(title, "It is time.", s.linkTo("/t/"+ReminderType+"/"+rec.ID))
	}
	return rang
}

// linkTo is a page of this workspace as a whole address, when the
// address this server listens on is known; the path alone otherwise.
func (s *Server) linkTo(path string) string {
	for _, k := range workspace.KnownWorkspaces() {
		if sameDir(k.Dir, s.app.Workspace.Dir) && k.Addr != "" {
			return "http://" + k.Addr + path
		}
	}
	return path
}
