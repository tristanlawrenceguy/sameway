package server

// MailInsecure lets a test sign in to its mail server without TLS.
func MailInsecure(on bool) { mailInsecure = on }
