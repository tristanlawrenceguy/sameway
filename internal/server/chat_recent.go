package server

import (
	"html/template"

	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// A conversation shows its latest messages, and the earlier ones when
// asked. Shown whole, a long chat put the newest reply and the box to type
// in past a screen reader's first minutes and past what an agent reading
// the page could hold: one chat was 115,000 characters, most of it old
// failures. The earlier messages are kept, and the way to them is one quiet
// link where they would be.

// EarlierPart is a conversation's messages before its latest.
const EarlierPart = "earlier"

// recentMessages is how many of the latest messages a conversation shows.
const recentMessages = 20

// recentFrom is the first of n messages shown: all of them when asked for.
func recentFrom(n int, all bool) int {
	if all || n <= recentMessages {
		return 0
	}
	return n - recentMessages
}

// earlierLink is the way to a conversation's earlier messages, landing on
// the first one shown now so the place is not lost.
func (s *Server) earlierLink(from, firstShown string) template.HTML {
	return template.HTML(`<p class="sw-quiet sw-chat__earlier">`) +
		s.part(ui.Link{Href: showURL(from, []string{EarlierPart}) + "#" + firstShown, Label: "Earlier messages"}) + `</p>`
}
