package server

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server/media"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// A feature in a package of its own (internal/server/media, ...) lists its
// routes as web.Routes and is handed its own service; the server holds the
// service and adds the routes to its one table (routes.go), so who may use
// each address, its tool and its reach are still read from one place.

// served is a feature's routes as the server's: each handler is given the
// feature's service, which of finds on the server.
func served[D any](rs []web.Route[D], of func(*Server) D) []route {
	out := make([]route, len(rs))
	for i, rt := range rs {
		h := rt.Handle
		out[i] = route{pattern: rt.Pattern, handle: func(s *Server, w http.ResponseWriter, r *http.Request) { h(of(s), w, r) },
			access: rt.Access, tool: rt.Tool, persons: rt.Persons, public: rt.Public, reach: rt.Reach}
	}
	return out
}

// mediaRoutes are files, recordings and meetings (internal/server/media).
func mediaRoutes() []route {
	return served(media.Routes, func(s *Server) *media.Service { return s.media })
}

// What media needs of the server beyond web.Deps (media.Deps).
var _ media.Deps = face{}

func (f face) Title(t *schema.Type, rec *store.Record) string { return f.title(t, rec) }
func (f face) Importable(t *schema.Type) bool                 { return f.importable(t) }
func (f face) Shown(r *http.Request) (always, here []string)  { return f.shown(r) }
func (f face) Fewer(page, key, what string, here []string) string {
	return f.fewer(page, key, what, here)
}
func (f face) Handler() http.Handler { return f.mux }

// Speech is how this server writes recordings down. Tests give their own.
type Speech = media.Speech

// UseSpeech sets how this server writes recordings down.
func (s *Server) UseSpeech(k Speech) { s.media.UseSpeech(k) }

// UseSpeakers sets how speakers are told apart, for tests.
func (s *Server) UseSpeakers(ready func() bool, diarize func(ctx context.Context, wav string, n int) ([]speech.Turn, error)) {
	s.media.UseSpeakers(ready, diarize)
}

// WriteDownInBackground has this server write recordings down itself
// (media/hostwrite.go).
func (s *Server) WriteDownInBackground() { s.media.WriteDownInBackground() }

// WriteDown has a recording written down on this computer: the
// assistant's write_down (chat/home_tools.go).
func (s *Server) WriteDown(id string) (string, error) { return s.media.WriteDown(id) }

// AddFile keeps a file and reads it, as the command line's add does.
func (s *Server) AddFile(ctx context.Context, src io.Reader, name, title string) (*store.Record, error) {
	return s.media.AddFile(ctx, src, name, title)
}

// UseAppTokens keeps the meeting apps' sign-ins in a file of its own.
func (s *Server) UseAppTokens(path string) { s.media.UseAppTokens(path) }

// FetchMeetings brings the transcripts of meetings that have ended from
// Teams and Zoom (media/meeting_fetch.go).
func (s *Server) FetchMeetings(now time.Time) { s.media.FetchMeetings(now) }
