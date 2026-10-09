package server

// MailInsecure lets a test sign in to its mail server without TLS.
func MailInsecure(on bool) { mailInsecure = on }

// SetUpSorting makes the starter tags and the action that sorts email.
func (s *Server) SetUpSorting() { s.setUpSorting() }
