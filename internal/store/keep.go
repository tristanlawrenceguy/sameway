package store

// What another package keeps for one open workspace lives on its store,
// not in a package map keyed by it. Two stores open side by side (two
// workspaces, or tests run in parallel) never share it, and it goes when
// the store closes rather than staying for as long as the program runs.
// records keeps each store's change listeners here (records.Listen), and
// search its way of finding by meaning (search.UseMeaning).

// Keep is what is kept here under key, made by mk the first time. A key
// is best a type of the keeping package's own, so no other package can
// reach it.
func (s *Store) Keep(key any, mk func() any) any {
	s.keeping.Lock()
	defer s.keeping.Unlock()
	if v, ok := s.kept[key]; ok {
		return v
	}
	if s.kept == nil {
		s.kept = map[any]any{}
	}
	v := mk()
	s.kept[key] = v
	return v
}

// Kept is what is kept here under key, or nil when nothing is.
func (s *Store) Kept(key any) any {
	s.keeping.Lock()
	defer s.keeping.Unlock()
	return s.kept[key]
}

// letGo drops everything kept, when the store closes.
func (s *Store) letGo() {
	s.keeping.Lock()
	defer s.keeping.Unlock()
	s.kept = nil
}
