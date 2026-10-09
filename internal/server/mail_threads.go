package server

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A conversation by email is a thread: each email names what it answers
// (In-Reply-To, References), and those names are what tie it to the email
// that started it. Nobody files a reply into its thread; the email says.

// threadOf is the email that started the thread the named messages are in,
// by id, or "" when none of them is here.
func (s *Server) threadOf(refs []string) string {
	if len(refs) == 0 {
		return ""
	}
	want := map[string]bool{}
	for _, r := range refs {
		want[strings.Trim(r, "<> ")] = true
	}
	emails, err := s.app.Store.List("email", store.ListOptions{OrderBy: "created_at"})
	if err != nil {
		return ""
	}
	for _, e := range emails {
		id, _ := e.Fields["message_id"].(string)
		if id == "" || !want[strings.Trim(id, "<> ")] {
			continue
		}
		if root, _ := e.Fields["thread"].(string); root != "" {
			return root
		}
		return e.ID
	}
	return ""
}
