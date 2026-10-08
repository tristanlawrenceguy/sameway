package server

import "context"

// MailInsecure lets a test sign in to its mail server without TLS.
func MailInsecure(on bool) { mailInsecure = on }

// TriageWaiting sorts what waits to be sorted, now.
func (s *Server) TriageWaiting() int { return s.triageWaiting(context.Background()) }
