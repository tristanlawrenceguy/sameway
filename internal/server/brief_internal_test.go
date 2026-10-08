package server

import "time"

// BriefIfDue is briefIfDue, for tests outside the package.
func (s *Server) BriefIfDue(now time.Time) bool { return s.briefIfDue(now) }

// ReviewIfDue is reviewIfDue, for tests outside the package.
func (s *Server) ReviewIfDue(now time.Time) bool { return s.reviewIfDue(now) }
