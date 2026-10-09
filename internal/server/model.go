package server

import "github.com/tristanlawrenceguy/sameway/internal/server/connect"

// connectRoutes are connecting the assistant to a model
// (internal/server/connect), which needs nothing of the server beyond
// web.Deps.
func connectRoutes() []route {
	return served(connect.Routes, func(s *Server) *connect.Service { return s.connect })
}
