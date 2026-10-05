package server

import "github.com/tristanlawrenceguy/sameway/internal/search"

// hitsApart tells apart results with one title and kind, two tasks called
// Call plumber, by ordinals: first of 2, second of 3. No raw field labels
// like "(due" or "(added" — those are database column names, not words for
// people. The rank function below gives each a number so identical titles
// can be distinguished.
func (s *Server) hitsApart(hits []search.Hit) []string {
	names := make([]string, len(hits))
	for i, h := range hits {
		names[i] = h.Title + " " + h.Type
	}
	return apart(names, func(i int) []string {
		// No field-based telling apart in search results.
		return nil
	}, func(i int) string {
		if rec, err := s.app.Store.Get(hits[i].Type, hits[i].ID); err == nil {
			return addedRank(rec)
		}
		return hits[i].ID
	})
}
