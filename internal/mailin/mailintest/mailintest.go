// Package mailintest is a mail server on this computer, for tests: a
// mailbox to sign in to without TLS, with mail put in it as it is sent.
package mailintest

import (
	"bytes"
	"net"
	"testing"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

// Server is a running mail server with one user.
type Server struct {
	Addr string
	user *imapmemserver.User
}

// Start runs a server until the test ends, with user's INBOX.
func Start(t testing.TB, user, password string) *Server {
	mem := imapmemserver.New()
	u := imapmemserver.NewUser(user, password)
	u.Create("INBOX", nil)
	mem.AddUser(u)
	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		InsecureAuth: true,
		Caps:         imap.CapSet{imap.CapIMAP4rev1: {}},
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return &Server{Addr: ln.Addr().String(), user: u}
}

// Folder makes a folder.
func (s *Server) Folder(name string) { s.user.Create(name, nil) }

// Put delivers a raw email to a folder.
func (s *Server) Put(t testing.TB, folder, raw string) {
	if _, err := s.user.Append(folder, literal{bytes.NewReader([]byte(raw)), int64(len(raw))}, &imap.AppendOptions{}); err != nil {
		t.Fatal(err)
	}
}

type literal struct {
	*bytes.Reader
	n int64
}

func (l literal) Size() int64 { return l.n }
