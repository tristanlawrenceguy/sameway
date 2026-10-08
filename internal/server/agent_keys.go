package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// An agent with a key of its own (chat/agent_keys.go) sends it as
// Authorization: Bearer sw_…, here and at /mcp. The key says who it is and
// what it may do; one that is not known, or was taken away, is refused
// rather than taken for no key at all. Without a key, a request from this
// computer is the owner's, as it always was.

// keyed makes a request with an agent's key that agent's, and keeps its
// changes to the agent's pace (records/pace.go); reading is never paced.
func (s *Server) keyed(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	r, ok := AgentKey(s.app, w, r)
	if !ok || r.Method == http.MethodGet || r.Method == http.MethodHead {
		return r, ok
	}
	if v := chat.VisitorOf(r.Context()); v.Agent {
		if wait := chat.Pace(v.Login); wait > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(wait/time.Second)+1))
			writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": apiError{Code: "slow_down", Message: chat.SlowDown(wait)}})
			return r, false
		}
	}
	return r, true
}

// AgentKey is keyed for any handler of the workspace's, MCP's too.
func AgentKey(a *app.App, w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	key := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if !strings.HasPrefix(key, chat.KeyPrefix) {
		return r, true
	}
	agent := chat.AgentByKey(a.Store, key)
	if agent == nil {
		w.Header().Set("WWW-Authenticate", `Bearer realm="sameway"`)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": apiError{Code: "unknown_key",
			Message: "that key is not one this workspace knows; it may have been taken away. The owner makes a new one with sameway agent add"}})
		return r, false
	}
	chat.Used(a.Store, agent)
	name, _ := agent.Fields["name"].(string)
	access, _ := agent.Fields["access"].(string)
	v := chat.Visitor{Name: name, Login: "agent:" + agent.ID, Access: access, Agent: true}
	return r.WithContext(chat.WithVisitor(r.Context(), v)), true
}

// WithAgentKeys is a handler that knows agents' keys, for what serves the
// workspace beside the pages: MCP over HTTP.
func WithAgentKeys(a *app.App, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r, ok := AgentKey(a, w, r); ok {
			next.ServeHTTP(w, r)
		}
	})
}
