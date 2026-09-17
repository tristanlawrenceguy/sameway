package server

import (
	"fmt"
	"html/template"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// paneLabel names a pane by what is in it, because a label that says
// History over a calendar is a label that lies. One block lends its own
// summary; more than one, or one with nothing to say, gets the side.
func paneLabel(side string, blocks []*store.Record) string {
	if len(blocks) == 1 {
		name, _ := blocks[0].Fields["component"].(string)
		props, _ := blocks[0].Fields["props"].(map[string]any)
		if summary := chat.Summarise(name, props); summary != "" {
			return summary
		}
	}
	return side
}

// pane renders one of the two full height columns. It collapses to a strip
// and remembers whether it was open, because a pane that reopens itself on
// every page load is a pane nobody closes twice.
func (s *Server) pane(side, label string, blocks []*store.Record, convo *conversation) template.HTML {
	if len(blocks) == 0 {
		return ""
	}
	var inner strings.Builder
	fmt.Fprintf(&inner, `<ol class="sw-plain sw-canvas sw-canvas--pane" aria-label="Blocks in the %s pane">`, side)
	for _, blk := range blocks {
		inner.WriteString(s.blockItem(blk, convo))
	}
	inner.WriteString(`</ol>`)
	body, err := s.app.Registry.RenderSlot("disclosure",
		map[string]any{"label": label, "open": true, "id": side + "-pane"},
		template.HTML(inner.String()))
	if err != nil {
		return ""
	}
	return body
}

// strip renders the blocks in the header or footer bar: a row, not a
// column, and no disclosure, because a bar is not something to fold.
func (s *Server) strip(bar string, blocks []*store.Record, convo *conversation) template.HTML {
	if len(blocks) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<ol class="sw-plain sw-canvas sw-canvas--strip" aria-label="Blocks in the %s">`, bar)
	for _, blk := range blocks {
		b.WriteString(s.blockItem(blk, convo))
	}
	b.WriteString(`</ol>`)
	return template.HTML(b.String())
}
