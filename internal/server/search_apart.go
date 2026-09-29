package server

import "github.com/tristanlawrenceguy/sameway/internal/search"

// hitsApart tells apart results with one title and kind, two tasks called
// Call plumber, by what their records say (apart.go).
func (s *Server) hitsApart(hits []search.Hit) []string {
	names := make([]string, len(hits))
	for i, h := range hits {
		names[i] = h.Title + " " + h.Type
	}
	return apart(names, func(i int) []string {
		h := hits[i]
		if t, ok := s.app.Types.Get(h.Type); ok {
			if rec, err := s.app.Store.Get(t.Name, h.ID); err == nil {
				return recordWays(t, rec)
			}
		}
		return []string{"id " + h.ID}
	})
}
