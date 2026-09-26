package mcp

import (
	"context"
	"net"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// What an agent connected over HTTP may do follows who it is for. At the
// computer itself, everything. Over the tailnet, Tailscale says whose it is
// and it may do what that person may in the workspace: the owner's own
// devices everything, someone who may edit (or host) what their own
// assistant may, someone who may look the reading tools. From anywhere
// else on the network, with only the token, there is nobody to go by, so
// it reads and does not change.

// readTools are what a connection from elsewhere may call.
var readTools = map[string]bool{"describe": true, "find_records": true, "get_record": true, "search": true}

type offKey struct{}

// offMachine says whether a request came from somewhere other than this
// computer.
func offMachine(r *http.Request) bool {
	if chat.VisitorOf(r.Context()).Login != "" {
		return true
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return true
	}
	ip := net.ParseIP(host)
	return ip == nil || !ip.IsLoopback()
}

// reach is what a connection may call: every tool, the tools a person's
// own assistant has (svc), or the reading ones.
type reach struct {
	all bool
	svc *chat.Service
}

func (s *Server) reachOf(ctx context.Context) reach {
	if off, _ := ctx.Value(offKey{}).(bool); !off {
		return reach{all: true}
	}
	v := chat.VisitorOf(ctx)
	switch {
	case v.Login == "":
		return reach{} // no one to go by
	case v.Owner():
		return reach{all: true}
	case v.Access == chat.Edit || v.Access == chat.Host:
		return reach{svc: s.App.Chat.For(v)}
	}
	return reach{}
}

// may says whether a connection may call a tool, and the chat service it
// runs through when it is one of the assistant's.
func (s *Server) may(ctx context.Context, name string) (bool, *chat.Service) {
	r := s.reachOf(ctx)
	if r.all {
		return true, s.App.Chat
	}
	for _, t := range s.listFor(ctx) {
		if t.Name == name {
			if r.svc != nil {
				return true, r.svc
			}
			return true, s.App.Chat
		}
	}
	return false, nil
}

// refusedOff is what a connection is told when it asks for what the one
// it is for may not do.
const refusedOff = "This connection may not do that. Over the tailnet it may do what its person may in the workspace; from elsewhere on the network, with only the token, it reads: describe, find_records, get_record and search."

// fromTailnet says whether Tailscale said who made a request: a person the
// workspace let in, or one of its owner's devices. Who they are is the
// authentication, so no token is asked of them.
func fromTailnet(r *http.Request) bool {
	return chat.VisitorOf(r.Context()).Login != ""
}

// listFor is the tools a connection may call.
func (s *Server) listFor(ctx context.Context) []tool {
	r := s.reachOf(ctx)
	all := s.tools()
	if r.all {
		return all
	}
	allowed := map[string]bool{}
	for k := range readTools {
		allowed[k] = true
	}
	if r.svc != nil {
		for _, t := range r.svc.Tools() {
			allowed[t.Name] = true
		}
	}
	var out []tool
	for _, t := range all {
		if allowed[t.Name] {
			out = append(out, t)
		}
	}
	return out
}
