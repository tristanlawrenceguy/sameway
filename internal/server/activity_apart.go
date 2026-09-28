package server

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// entriesApart tells apart log entries shown together that say the same,
// "created task Call plumber" twice, whose links and Undo buttons would
// share a name (apart.go): the way the record's own list tells it apart
// while it exists, then by their time, then by their id.
func (s *Server) entriesApart(recs []*store.Record) []string {
	names := make([]string, len(recs))
	for i, r := range recs {
		w := chat.Say(r.Fields, true)
		names[i] = strings.Join([]string{w.Action, w.Target, w.Detail}, " ")
	}
	return apart(names, func(i int) []string {
		r := recs[i]
		ways := make([]string, 4)
		target, _ := r.Fields["target"].(string)
		id, _ := r.Fields["target_id"].(string)
		if t, ok := s.app.Types.Get(target); ok && id != "" {
			if rec, err := s.app.Store.Get(t.Name, id); err == nil {
				ways = recordWays(t, rec)
			}
		}
		return append(ways, "at "+momentWords(r.CreatedAt), "entry "+shortID(r.ID), "entry "+r.ID)
	})
}
