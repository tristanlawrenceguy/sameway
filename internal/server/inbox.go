package server

import "github.com/tristanlawrenceguy/sameway/internal/server/inbox"

// inboxRoutes are email in and what waits to be sorted
// (internal/server/inbox), which needs nothing of the server that media
// and exchange do not already: files, the parts of a page and the clock.
func inboxRoutes() []route {
	return served(inbox.Routes, func(s *Server) *inbox.Service { return s.inbox })
}

var _ inbox.Deps = face{}

// SetUpSorting makes the starter tags and the actions that sort what comes
// in, as connecting email does; a test sorts without a mailbox.
func (s *Server) SetUpSorting() { s.inbox.SetUpSorting() }
