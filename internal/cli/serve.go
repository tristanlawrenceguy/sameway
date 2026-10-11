package cli

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
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
	return http.NewCrossOriginProtection().Handler(mux)
}
