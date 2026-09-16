package server

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Tabs are canvases. Home is the first, at /; every canvas record is one
// more tab at /c/<id>, with blocks of its own. Switching tabs is a page
// navigation, like everything else here, so the bar is a list of links and
// needs no script; and it only appears once there is a second tab, because
// a bar with one tab is chrome saying nothing.

// tabBar renders the list of canvases with the current one marked, or
// nothing when Home is the only one.
func (s *Server) tabBar(current string) template.HTML {
	canvases := s.app.Chat.Canvases()
	if len(canvases) < 2 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<nav class="sw-tabs" aria-label="Canvases"><ul class="sw-plain sw-tabs__list">`)
	for _, c := range canvases {
		fmt.Fprintf(&b, `<li>%s</li>`, s.component("link", map[string]any{
			"href": c.Path, "label": c.Name, "current": c.ID == current,
		}))
	}
	b.WriteString(`</ul></nav>`)
	return template.HTML(b.String())
}

// seedChat puts a conversation on an empty canvas, so a new workspace and a
// new tab both open on the chat, and a cleared canvas recovers one.
func (s *Server) seedChat(canvas string) {
	blocks := chat.OnCanvas(s.canvasBlocks(), canvas)
	if len(blocks) > 0 {
		return
	}
	s.app.Store.Create(chat.BlockType, s.app.Chat.BlockFields(map[string]any{
		"component": chat.ComponentName, "props": map[string]any{}, "position": 0, "span": 12, "canvas": canvas,
	}))
}

// canvasOf is the tab a block is on.
func canvasOf(fields map[string]any) string {
	on, _ := fields["canvas"].(string)
	return on
}
