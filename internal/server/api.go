package server

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server/api"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// agentAPIRoutes are the JSON API for agents (internal/server/api).
func agentAPIRoutes() []route {
	return served(api.Routes, func(s *Server) *api.Service { return s.api })
}

// What the API needs of the server beyond web.Deps (api.Deps): the
// assistant as the one asking may use it, what a log entry says, and the
// conversation a new tab starts with.
var _ api.Deps = face{}

func (f face) ChatFor(r *http.Request) *chat.Service { return f.chatFor(r) }
func (f face) ActivityFacts(e *store.Record) []any   { return f.activityFacts(e) }
func (f face) SeedChat(canvas string)                { f.seedChat(canvas) }
