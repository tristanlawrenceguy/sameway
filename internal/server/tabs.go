package server

import (
	"html/template"

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
	var items []any
	for _, c := range canvases {
		items = append(items, map[string]any{"href": c.Path, "label": c.Name, "current": c.ID == current})
	}
	return s.component("tabs", map[string]any{"label": "Canvases", "items": items})
}

// tabName is what a tab's page is called, in its window title and its
// heading: its own name once there are tabs, so Home, Garden and Money are
// told apart when they arrive, and Canvas while there is only the one.
func (s *Server) tabName(canvas string) string {
	canvases := s.app.Chat.Canvases()
	if len(canvases) < 2 {
		return "Canvas"
	}
	for _, c := range canvases {
		if c.ID == canvas {
			return c.Name
		}
	}
	return "Canvas"
}

// seedChat puts a conversation on an empty canvas, so a new workspace and a
// new tab both open on the chat, and a cleared canvas recovers one.
func (s *Server) seedChat(canvas string) {
	blocks := chat.OnCanvas(s.canvasBlocks(), canvas)
	if len(blocks) > 0 {
		return
	}
	s.app.Store.Create(chat.BlockType, s.app.Chat.BlockFields(map[string]any{
		"component": chat.ComponentName, "props": map[string]any{}, "position": 0, "span": 12, "canvas": canvas, "actor": "system", "created_by": "system",
	}))
	// Search starts in the header of the first tab, where a person reaches
	// for it on every page. It is a block like any other: move it, shrink
	// it to an icon, or remove it, and the search page is still there.
	if canvas == "" {
		s.app.Store.Create(chat.BlockType, s.app.Chat.BlockFields(map[string]any{
			"component": "search", "props": map[string]any{}, "position": 0, "span": 4, "region": "header", "frame": "bare", "canvas": canvas, "actor": "system", "created_by": "system",
		}))
	}
}

// canvasOf is the tab a block is on.
func canvasOf(fields map[string]any) string {
	on, _ := fields["canvas"].(string)
	return on
}
