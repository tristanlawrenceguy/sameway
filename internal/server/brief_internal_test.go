package server

import "time"

// BriefIfDue is briefIfDue, for tests outside the package.
func (s *Server) BriefIfDue(now time.Time) bool { return s.briefIfDue(now) }
