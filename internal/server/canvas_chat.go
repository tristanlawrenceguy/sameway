package server

import (
	"html/template"
	"log"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// chatBlock renders a chat component with the live conversation inside it.
func (s *Server) chatBlock(blk *store.Record, convo *conversation) template.HTML {
	props, _ := blk.Fields["props"].(map[string]any)
	if props == nil {
		props = map[string]any{}
	}
	out, err := s.app.Registry.RenderSlot(records.ComponentName, props, convo.Body)
	if err != nil {
		log.Printf("render chat: %v", err)
		return s.component("alert", map[string]any{"kind": "danger", "message": "The conversation could not be shown. Reload the page to try again."})
	}
	return out
}
