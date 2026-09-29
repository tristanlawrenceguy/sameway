package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Every change a client makes is logged as that agent's, by the name it
// gives on initialize (clientInfo: its title, or its name, such as
// claude-code, said as Claude Code), so the person can tell Claude Code
// from ChatGPT, and either from themselves and from the assistant in the
// app. Over stdio the connection lasts as long as the process; over HTTP
// the client is given an Mcp-Session-Id and sends it back, as Streamable
// HTTP clients do. A client that says nothing, and sends no
// X-Sameway-Agent header, is "An agent".

// conn is one client: what it calls itself.
type conn struct {
	mu   sync.Mutex
	name string
	// fresh says it introduced itself during this request, so an HTTP
	// client is given a session to be known by after.
	fresh bool
}

type connKey struct{}

func withConn(ctx context.Context, c *conn) context.Context {
	return context.WithValue(ctx, connKey{}, c)
}

// connOf is the client a request is from; one of nobody in particular
// when the context carries none.
func connOf(ctx context.Context) *conn {
	if c, ok := ctx.Value(connKey{}).(*conn); ok {
		return c
	}
	return &conn{}
}

// introduce takes the client's name from initialize's params. A name the
// connection already has from its X-Sameway-Agent header stays: that is
// what the person chose to call it.
func (c *conn) introduce(params json.RawMessage) {
	var p struct {
		ClientInfo struct {
			Name  string `json:"name"`
			Title string `json:"title"`
		} `json:"clientInfo"`
	}
	json.Unmarshal(params, &p)
	name := p.ClientInfo.Title
	if name == "" {
		name = p.ClientInfo.Name
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.name == "" {
		c.name = chat.AgentName(name)
	}
	c.fresh = true
}

func (c *conn) agent() chat.Agent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return chat.Agent{Name: c.name, Through: chat.ThroughMCP}
}

// forAgent is the service a tool runs through, as the client's: the
// assistant in the app stays the assistant.
func (s *Server) forAgent(ctx context.Context, svc *chat.Service) *chat.Service {
	if s.Assistant {
		return svc
	}
	return svc.ByAgent(connOf(ctx).agent())
}

// sessions are the HTTP clients known by the session id each was given.
type sessions struct {
	mu   sync.Mutex
	byID map[string]*conn
}

// maxSessions bounds what is remembered: a client forgotten is asked
// nothing again, it only reads as "An agent" until it initializes anew.
const maxSessions = 256

// keep gives a client that introduced itself during this request a
// session id, and remembers it by that.
func (ss *sessions) keep(c *conn) string {
	c.mu.Lock()
	fresh := c.fresh
	c.fresh = false
	c.mu.Unlock()
	if !fresh {
		return ""
	}
	raw := make([]byte, 16)
	rand.Read(raw)
	id := hex.EncodeToString(raw)
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if ss.byID == nil {
		ss.byID = map[string]*conn{}
	}
	if len(ss.byID) >= maxSessions {
		for k := range ss.byID {
			delete(ss.byID, k)
			break
		}
	}
	ss.byID[id] = c
	return id
}

// connFor is the client an HTTP request is from: the one its session id
// names, or a new one named by its X-Sameway-Agent header, if any.
func (s *Server) connFor(r *http.Request) *conn {
	if id := r.Header.Get("Mcp-Session-Id"); id != "" {
		s.sessions.mu.Lock()
		c := s.sessions.byID[id]
		s.sessions.mu.Unlock()
		if c != nil {
			return c
		}
	}
	// An agent with a key is who its key says, whatever it calls itself.
	if v := chat.VisitorOf(r.Context()); v.Agent {
		return &conn{name: v.Name}
	}
	return &conn{name: chat.AgentName(r.Header.Get("X-Sameway-Agent"))}
}
