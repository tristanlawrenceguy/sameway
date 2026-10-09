// Package connect is connecting the assistant to an AI model: the card
// that says what is wrong and offers what is on this computer, each way
// of having a model told as its own story, a pasted key checked and kept
// in the person's own settings, a free model fetched through Ollama, and
// the AI apps a person already uses connected to the workspace. The
// server holds one Service and adds its Routes to the one route table.
package connect

import (
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Service is connecting the assistant for one workspace: whether it can
// reach its model, as last looked.
type Service struct {
	web.Deps
	app   *app.App
	model modelState // connect.go
}

// New is connecting the assistant for the workspace the server serves.
func New(d web.Deps) *Service { return &Service{Deps: d, app: d.App()} }

// keys is the file a pasted key is kept in (workspace.Machine).
func (s *Service) keys() llm.Keys { return llm.Keys(s.app.Workspace.Machine.Keys) }
