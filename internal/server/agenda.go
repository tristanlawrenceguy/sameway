package server

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server/agenda"
	"github.com/tristanlawrenceguy/sameway/internal/server/inbox"
)

// agendaRoutes are Today, the morning brief and the weekly review
// (internal/server/agenda).
func agendaRoutes() []route {
	return served(agenda.Routes, func(s *Server) *agenda.Service { return s.agenda })
}

// What the agenda needs of the server beyond web.Deps (agenda.Deps): the
// assistant's questions and what came in, both on Today, and a ring
// beyond the page for the brief and the review.
var _ agenda.Deps = face{}

func (f face) Questions(list []chat.Waiting) string { return f.questionSection(list) } // proposal.go
func (f face) Inbox() *inbox.Service                { return f.inbox }
func (f face) ForgetModel()                         { f.connect.ForgetModel() }
func (f face) CanNotify() bool                      { return f.notify != nil }
func (f face) Notify(title, text, path string)      { go f.notify(title, text, f.linkTo(path)) } // ring.go

// BriefIfDue sends today's brief when it is due, and says whether it did;
// ReviewIfDue the same for the weekly review. For tests outside the
// package; the runner has them as jobs (jobs.go).
func (s *Server) BriefIfDue(now time.Time) bool  { return s.agenda.BriefIfDue(now) }
func (s *Server) ReviewIfDue(now time.Time) bool { return s.agenda.ReviewIfDue(now) }
