package server

import (
	"slices"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/render"
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

// pickHabits is the habits a tracker names, by id or by name however it
// is written, in the order named; a name that is no habit's makes it say
// so, with the one likely meant and the habits there are.
func pickHabits(recs []*store.Record, named []string) ([]*store.Record, string) {
	var out []*store.Record
	var names []string
	for _, rec := range recs {
		if n, _ := rec.Fields["name"].(string); n != "" {
			names = append(names, n)
		}
	}
	for _, want := range named {
		var found *store.Record
		for _, rec := range recs {
			n, _ := rec.Fields["name"].(string)
			if rec.ID == want || strings.EqualFold(strings.TrimSpace(n), strings.TrimSpace(want)) {
				found = rec
				break
			}
		}
		if found == nil {
			if len(names) == 0 {
				return nil, "there is no habit " + want + ", and no habit yet; create_record a habit first"
			}
			said := "there is no habit " + want
			if near := render.Nearest(want, names); near != "" {
				said += " (did you mean " + near + "?)"
			}
			return nil, said + "; the habits are " + strings.Join(names, ", ")
		}
		if !slices.Contains(out, found) {
			out = append(out, found)
		}
	}
	return out, ""
}
