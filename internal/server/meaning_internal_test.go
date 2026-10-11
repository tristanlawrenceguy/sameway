package server

// MeaningFetching has search by meaning being fetched, done of total
// bytes here, for tests outside the package.
func (s *Server) MeaningFetching(done, total int64) {
	st := s.meaningState()
	st.Lock()
	st.fetching, st.done, st.total = true, done, total
	st.Unlock()
}

// MeaningLine is meaningLine, for tests outside the package.
func (s *Server) MeaningLine() string { return s.meaningLine() }
