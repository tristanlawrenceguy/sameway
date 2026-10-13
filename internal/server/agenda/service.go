// Package agenda is what is on: Today (what is late, due and on today,
// what waits for the person, three to start with when there is a lot, and
// a box for everything in their head), the morning brief that says it as
// one notification, and the weekly review with the day it says it is
// ready. The server holds one Service and adds its Routes to the one route
// table.
package agenda

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/server/inbox"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Deps is what the agenda needs of the server: what every handler does
// (web.Deps), and a few things of its own.
type Deps interface {
	web.Deps
	// Title is a record's name, as its page says it; H24 whether times
	// are read on the 24-hour clock.
	Title(t *schema.Type, rec *store.Record) string
	H24() bool
	// ChatFor is the assistant as the one asking may use it
	// (server/access.go); ForgetModel has the connect card look again
	// after a turn failed.
	ChatFor(r *http.Request) *chat.Service
	ForgetModel()
	// Questions are what the assistant asks, as under the conversation
	// (server/proposal.go); Inbox is what came in to sort.
	Questions(list []chat.Waiting) string
	Inbox() *inbox.Service
	// CanNotify says whether a ring reaches beyond the page, and Notify
	// sends one leading to a page of this workspace (server/ring.go).
	CanNotify() bool
	Notify(title, text, path string)
}

// Service is the agenda of one workspace.
type Service struct {
	Deps
	app *app.App
}

// New is the agenda of the workspace the server serves.
func New(d Deps) *Service { return &Service{Deps: d, app: d.App()} }
