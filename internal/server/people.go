package server

import (
	stdcmp "cmp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// whoDid is who made a change in the log when it was someone other than
// the owner of this computer's copy: their name and their colour. "" is
// the owner, who reads as "You". A person made a change here from their
// own device, or on another computer that hosts the workspace, and the
// entry came by sync.
func (s *Server) whoDid(entry *store.Record) (string, int) {
	if entry.Fields["actor"] != "human" {
		return "", 0
	}
	login, _ := entry.Fields["by_login"].(string)
	by, _ := entry.Fields["by"].(string)
	if login == "" && by == "" || login != "" && strings.EqualFold(login, s.app.Chat.Owner.Login) {
		return "", 0
	}
	name := by
	if name == "" {
		if p := s.app.Chat.PersonByEmail(login); p != nil {
			name, _ = p.Fields["name"].(string)
		}
	}
	if name == "" {
		name, _, _ = strings.Cut(login, "@")
	}
	return name, chat.PersonColour(stdcmp.Or(login, by))
}

// changedBy is, for each block someone other than the owner changed
// lately, that person's colour: the latest change to a block says whose
// colour it glows in. The log is read newest first and only so far back,
// since a glow only lasts for the latest changes.
func (s *Server) changedBy() map[string]int {
	out := map[string]int{}
	entries, err := s.app.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 40})
	if err != nil {
		return out
	}
	seen := map[string]bool{}
	for _, e := range entries {
		id, _ := e.Fields["target_id"].(string)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		if _, person := s.whoDid(e); person > 0 {
			out[id] = person
		}
	}
	return out
}
