package server

import "github.com/tristanlawrenceguy/sameway/internal/search"

// hitsApart tells apart results of one title and kind, two tasks called
// Call plumber, the way their list does (design/foundations/names.md): by
// their day as their glance says it, then when each was added, and last by
// where each comes, numbered as their list numbers them. Results of one
// title and different kinds are told apart by their kind, which a result
// already says. Search once numbered them all, the first of 2 where the
// list said due Friday, after "due" was taken for a stored name; it is the
// word a person uses.
func (s *Server) hitsApart(hits []search.Hit) []string {
	names := make([]string, len(hits))
	for i, h := range hits {
		names[i] = h.Title + " " + h.Type
	}
	return apart(names, func(i int) []string {
		if t, ok := s.app.Types.Get(hits[i].Type); ok {
			if rec, err := s.app.Store.Get(t.Name, hits[i].ID); err == nil {
				return s.recordWays(t, rec)
			}
		}
		return nil
	}, func(i int) string {
		if rec, err := s.app.Store.Get(hits[i].Type, hits[i].ID); err == nil {
			return addedRank(rec)
		}
		return hits[i].ID
	})
}
