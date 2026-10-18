// Package api is the JSON API for agents: what the pages do, as JSON. The
// description of the workspace, a record and a list of them a page or a
// few fields at a time, writes logged as the agent's who made them, a
// block held to what the assistant's are, the assistant's tools over
// REST, a chat turn, the changes since an agent last looked, and the
// background jobs. The server holds one Service and adds its Routes to
// the one route table; the API's routes that belong to another feature
// (look, prose, types, actions, search, workspaces) stay with it.
package api

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server/media"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Deps is what the API needs of the server: what every handler does
// (web.Deps), and a few things of its own.
type Deps interface {
	web.Deps
	// ChatFor is the assistant as the one asking may use it: the owner's,
	// or the tools someone let in may have (server/access.go).
	ChatFor(r *http.Request) *chat.Service
	// Title is a record's name, as its page says it.
	Title(t *schema.Type, rec *store.Record) string
	// ActivityFacts are what an entry in the log says, as its page does.
	ActivityFacts(e *store.Record) []any
	// SeedChat gives a tab no one has opened its conversation, as opening
	// it would (server/tabs.go).
	SeedChat(canvas string)
	// Media is files: a file sent inside JSON is kept and read as one.
	Media() *media.Service
	// PageRoute says method and path are one of the server's own routes,
	// outside /api (api.go's not found).
	PageRoute(method, path string) bool
}

// Service is the API of one workspace, and those waiting on its changes.
type Service struct {
	Deps
	app    *app.App
	logged signal // a change logged here, for those waiting on /api/changes; changes.go
}

// New is the API of the workspace the server serves.
func New(d Deps) *Service { return &Service{Deps: d, app: d.App()} }

// Logged wakes everyone waiting on /api/changes: a change was logged here.
func (s *Service) Logged() { s.logged.fire() }
