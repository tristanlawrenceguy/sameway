package server

import (
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

func hasTag(rec *store.Record, tags []string) bool {
	have, _ := rec.Fields["tags"].([]any)
	for _, t := range have {
		for _, want := range tags {
			if s, _ := t.(string); strings.EqualFold(s, want) {
				return true
			}
		}
	}
	return false
}

// habitTags is every tag the habits carry, so a tracker asked for one
// that none has can say which they do.
func (s *Server) habitTags() []string {
	recs, _ := s.app.Store.List(HabitType, store.ListOptions{})
	seen := map[string]bool{}
	var out []string
	for _, rec := range recs {
		for _, tag := range strs(rec.Fields["tags"]) {
			if !seen[tag] {
				seen[tag] = true
				out = append(out, tag)
			}
		}
	}
	sort.Strings(out)
	return out
}
