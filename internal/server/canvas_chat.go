package server

import (
	"html/template"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// chatBlock renders a chat component with the live conversation inside it.
func (s *Server) chatBlock(blk *store.Record, convo *conversation) template.HTML {
	props, _ := blk.Fields["props"].(map[string]any)
	if props == nil {
		props = map[string]any{}
	}
	// The chat is a block, so its menu can also place the block.
	body := strings.Replace(string(convo.Body), `<span class="sw-chat__place"></span>`, string(s.placeMenu(blk, convo.From)), 1)
	out, err := s.app.Registry.RenderSlot(chat.ComponentName, props, template.HTML(body))
	if err != nil {
		return s.component("alert", map[string]any{"kind": "danger", "message": "Could not render the conversation: " + err.Error()})
	}
	return out
}
