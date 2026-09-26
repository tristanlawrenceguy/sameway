package server

import "net/http"

// togetherRoutes are the ways people who share a workspace work together:
// choosing between two versions written at once (clash.go), and saying
// they have caught up on what others did (since.go).
func (s *Server) togetherRoutes(m *http.ServeMux) {
	m.HandleFunc("POST /clash/{id}/use", s.clashUse)
	m.HandleFunc("POST /clash/{id}/keep", s.clashKeep)
}
