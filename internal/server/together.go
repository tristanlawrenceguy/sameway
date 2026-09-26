package server

import "net/http"

// togetherRoutes are the ways people who share a workspace work together:
// choosing between two versions written at once (clash.go), and saying
// they have caught up on what others did (since.go). A record arriving
// from another computer made out for this one's owner tells them
// (foryou.go).
func (s *Server) togetherRoutes(m *http.ServeMux) {
	s.app.Store.AfterSync = s.forYou
	m.HandleFunc("POST /clash/{id}/use", s.clashUse)
	m.HandleFunc("POST /clash/{id}/keep", s.clashKeep)
	m.HandleFunc("POST /since/seen", s.sinceSeen)
}
