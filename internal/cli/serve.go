package cli

import (
	"net"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// Handler is the whole server: the pages and the API, and MCP over HTTP at
// /mcp, behind the workspace's token, or for someone Tailscale says the
// workspace let in; from anywhere but this computer it reads only.
func Handler(a *app.App, token string) http.Handler {
	return HandlerFor(a, token, server.New(a))
}

// HandlerFor is Handler around a server already made, so the command
// line can start its ringing first.
//
// Every way of serving (serve, open, the Wi-Fi and the tailnet) goes
// through here, so this is where another site's page is stopped: a
// browser visiting it could otherwise post forms to this computer
// (CSRF). A write from a browser must come from Sameway's own pages; the
// command line, MCP and API clients send neither Origin nor
// Sec-Fetch-Site and pass, as does a phone's share (the browser's own).
func HandlerFor(a *app.App, token string, s *server.Server) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/mcp", server.WithAgentKeys(a, mcp.Bearer(token, &mcp.Server{App: a, Version: update.Version})))
	mux.Handle("/", s)
	return http.NewCrossOriginProtection().Handler(ownNames(token, mux))
}

// ownNames refuses a request sent to a name that is not this computer's.
// Another site can point its own name at this computer (DNS rebinding):
// its page is then same-origin here, and a request from this computer is
// the owner's, so it could read the workspace and search its folders. A
// tunnel (cloudflared, ngrok) brings the internet in under its own name
// the same way. So a name from elsewhere is let in only with a key that
// says who it is: an agent's key, checked after this, or MCP's token at
// /mcp.
func ownNames(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		keyed := strings.HasPrefix(key, records.KeyPrefix) || r.URL.Path == "/mcp" && token != "" && key == token
		if !ownName(r.Host) && !keyed {
			http.Error(w, "Sameway opens at this computer's own address, such as http://localhost, or its name on the Wi-Fi or the tailnet. An agent reaching it by another name sends its key.", http.StatusMisdirectedRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ownName is whether a Host names this computer: an address, localhost,
// a name with no dots (the computer's own, or MagicDNS's short one), or
// one on the Wi-Fi (.local) or the tailnet (.ts.net). Nobody else can
// make a browser take those to this computer.
func ownName(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.ToLower(strings.Trim(host, "[]")), ".")
	if net.ParseIP(host) != nil || !strings.Contains(host, ".") {
		return true
	}
	for _, own := range []string{".localhost", ".local", ".ts.net"} {
		if strings.HasSuffix(host, own) {
			return true
		}
	}
	return false
}
