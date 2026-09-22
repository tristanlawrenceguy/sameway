package cli

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/mcp"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// Handler is the whole server: the pages and the API, and MCP over HTTP at
// /mcp for a client elsewhere, behind the workspace's token.
func Handler(a *app.App, token string) http.Handler {
	return HandlerFor(a, token, server.New(a))
}

// HandlerFor is Handler around a server already made, so the command
// line can start its ringing first.
func HandlerFor(a *app.App, token string, s *server.Server) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.Bearer(token, &mcp.Server{App: a, Version: update.Version}))
	mux.Handle("/", s)
	return mux
}
