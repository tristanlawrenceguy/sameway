// Package inbox is what comes in and waits to be sorted: email read from
// the person's mailbox, each service's own story for connecting it, an
// email's conversation and whose turn it is to reply, and on Today what
// came in to sort, the tags an action gave to check and what it suggested
// to make. The server holds one Service and adds its Routes to the one
// route table.
package inbox

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server/media"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Deps is what inbox needs of the server: what every handler does
// (web.Deps), and a few things of its own.
type Deps interface {
	web.Deps
	// Media is files: what an email brings is kept and read as one.
	Media() *media.Service
	// Shown and Fewer are the parts of a page that are off until there is
	// a reason (server/parts.go): which this view shows, and the way back
	// out of one the address opened.
	Shown(r *http.Request) (always, here []string)
	Fewer(page, key, what string, here []string) string
	// H24 is whether times are read on the 24-hour clock.
	H24() bool
}

// Service is the inbox of one workspace.
type Service struct {
	Deps
	app *app.App
}

// New is the inbox of the workspace the server serves.
func New(d Deps) *Service { return &Service{Deps: d, app: d.App()} }

// keys is the file the mailbox's password is kept in (workspace.Machine).
func (s *Service) keys() llm.Keys { return llm.Keys(s.app.Workspace.Machine.Keys) }

// Today is what came in, on Today, from the list of what waits: To sort,
// Your turn to reply, Suggested for you and Tags to check, in that order.
func (s *Service) Today(list []chat.Waiting) string {
	return s.mailSection(list) + s.turnSection(list) + s.suggestedSection(list, s.sortingRefs()) + s.tagSection(list, s.sortingRefs()) // mail_sort.go, turns_today.go, suggested.go, tags_today.go
}
