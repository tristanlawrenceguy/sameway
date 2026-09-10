package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// canvasPage is the home page: the canvas, full width. Everything on it is a
// block, including the conversation, so any of it can be moved, restyled, or
// removed. The conversation also lives at /chat, so removing its block never
// locks anyone out.
func (s *Server) canvasPage(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Chat.Available(); err != nil {
		s.page(w, r, "Canvas", s.component("alert", map[string]any{"kind": "danger", "title": "This workspace is incomplete", "message": err.Error()}), pageOptions{})
		return
	}
	s.seedChat()
	blocks, err := s.app.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
	if err != nil {
		s.fail(w, err)
		return
	}
	convo, err := s.conversation("/")
	if err != nil {
		s.fail(w, err)
		return
	}

	var b strings.Builder
	if convo.Notice != "" {
		b.WriteString(string(convo.Notice))
	}
	var main, side []*store.Record
	for _, blk := range blocks {
		if str(blk.Fields["region"], "main") == "side" {
			side = append(side, blk)
		} else {
			main = append(main, blk)
		}
	}
	// A workspace with nothing but the conversation shows just that, in the
	// middle of the page, the way every other assistant opens. Everything
	// else arrives because someone asked for it.
	solo := len(side) == 0 && len(main) == 1 && main[0].Fields["component"] == chat.ComponentName

	if len(blocks) == 0 {
		b.WriteString(`<p class="sw-empty">This canvas is empty. The conversation is at <a href="/chat">/chat</a>, and anything you ask for there appears here.</p>`)
	} else {
		fmt.Fprintf(&b, `<div class="sw-page" data-layout="%s">`, layoutName(solo, len(side) > 0))
		b.WriteString(`<ol class="sw-plain sw-canvas" aria-label="Canvas">`)
		for _, blk := range main {
			b.WriteString(s.blockItem(blk, convo))
		}
		b.WriteString(`</ol>`)
		if len(side) > 0 {
			b.WriteString(s.sidePane(side, convo))
		}
		b.WriteString(`</div>`)
	}
	if convo.Activity != "" {
		b.WriteString(`<div class="sw-activity">` + string(convo.Activity) + `</div>`)
	}
	opts := pageOptions{JSONURL: "/api/block"}
	if convo.LatestID != "" {
		opts.Focus, opts.FocusLabel = convo.LatestID, "Skip to latest message"
	}
	opts.QuietTitle = solo
	s.page(w, r, "Canvas", template.HTML(b.String()), opts)
}

func layoutName(solo, hasSide bool) string {
	switch {
	case solo:
		return "solo"
	case hasSide:
		return "split"
	}
	return "wide"
}

// sidePane holds what a person glances at rather than works in. It collapses
// to a strip and remembers whether it was open, because a pane that reopens
// itself on every page load is a pane nobody closes twice.
func (s *Server) sidePane(blocks []*store.Record, convo *conversation) string {
	var inner strings.Builder
	inner.WriteString(`<ol class="sw-plain sw-canvas sw-canvas--side" aria-label="Side pane blocks">`)
	for _, blk := range blocks {
		inner.WriteString(s.blockItem(blk, convo))
	}
	inner.WriteString(`</ol>`)
	body, err := s.app.Registry.RenderSlot("disclosure",
		map[string]any{"label": "Side pane", "open": true, "id": "side-pane"},
		template.HTML(inner.String()))
	if err != nil {
		return ""
	}
	return `<aside class="sw-aside" aria-label="Side pane">` + string(body) + `</aside>`
}

// blockItem renders one canvas block: the component, its span, its
// provenance rail, and its quiet control bar.
func (s *Server) blockItem(blk *store.Record, convo *conversation) string {
	v := s.canvasBlock(blk, convo)
	body := v.HTML
	if v.Component == chat.ComponentName {
		body = s.chatBlock(blk, convo)
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<li class="sw-block sw-reveal" data-block-id="%s" data-block-component="%s" data-actor="%s" data-frame="%s" data-tone="%s"`,
		v.ID, v.Component, v.Actor, v.Frame, v.Tone)
	if v.Changed != "" {
		fmt.Fprintf(&b, ` data-changed="%s"`, v.Changed)
	}
	fmt.Fprintf(&b, ` style="--sw-span: %d; view-transition-name: block-%s; view-transition-class: sw-vt-item">`, v.Span, v.ID)
	// Provenance costs nothing on screen and is complete in the
	// accessibility tree. Sighted people got it from the glow when it
	// happened, and can get it again from the activity log.
	fmt.Fprintf(&b, `<p class="sw-visually-hidden">%s</p>`, template.HTMLEscapeString(v.Provenance))
	b.WriteString(string(body))
	fmt.Fprintf(&b, `<div class="sw-bar sw-quiet"><form method="post" action="/canvas/%s/delete">%s</form></div></li>`,
		v.ID, v.Remove)
	return b.String()
}

// chatBlock renders a chat component with the live conversation inside it.
func (s *Server) chatBlock(blk *store.Record, convo *conversation) template.HTML {
	props, _ := blk.Fields["props"].(map[string]any)
	if props == nil {
		props = map[string]any{}
	}
	out, err := s.app.Registry.RenderSlot(chat.ComponentName, props, convo.Body)
	if err != nil {
		return s.component("alert", map[string]any{"kind": "danger", "message": "Could not render the conversation: " + err.Error()})
	}
	return out
}

// seedChat puts a conversation on an empty canvas, so a new workspace has
// somewhere to talk and a cleared canvas recovers one.
func (s *Server) seedChat() {
	if n, err := s.app.Store.Count(chat.BlockType); err != nil || n > 0 {
		return
	}
	s.app.Store.Create(chat.BlockType, s.app.Chat.BlockFields(map[string]any{
		"component": chat.ComponentName, "props": map[string]any{}, "position": 0, "span": 12,
	}))
}

// canvasBlock gathers everything the page needs about one block.
func (s *Server) canvasBlock(b *store.Record, convo *conversation) canvasBlock {
	name, _ := b.Fields["component"].(string)
	props, _ := b.Fields["props"].(map[string]any)
	actor, _ := b.Fields["actor"].(string)
	if actor == "" {
		actor = "assistant"
	}
	createdBy, _ := b.Fields["created_by"].(string)
	if createdBy == "" {
		createdBy = actor
	}
	span := 6
	if v, ok := b.Fields["span"].(int64); ok && v >= 1 && v <= 12 {
		span = int(v)
	}
	who := map[string]string{"human": "you", "assistant": "the assistant"}
	provenance := "Added by " + who[createdBy] + "."
	if actor != createdBy {
		provenance = "Added by " + who[createdBy] + ", edited by " + who[actor] + "."
	}

	// Glow only for what changed in the exchange just finished, or in the
	// last few seconds, so a marker never outlives the change it reports.
	changed := ""
	if inLastTurn(b.CreatedAt, convo) {
		changed = "added"
	} else if inLastTurn(b.UpdatedAt, convo) {
		changed = "updated"
	}
	// Removing is the one thing worth a control of its own. Anything else a
	// person wants changed, they ask for, which is faster than any form and
	// is the whole point of having an assistant on the page.
	return canvasBlock{
		ID: b.ID, Component: name, Actor: actor, Changed: changed, Span: span,
		Frame: str(b.Fields["frame"], "card"), Tone: str(b.Fields["tone"], "none"),
		Provenance: provenance,
		HTML:       s.component(name, props),
		Remove:     s.component("button", map[string]any{"label": "Remove", "context": name, "type": "submit", "variant": "quiet"}),
	}
}

// turnIsFresh bounds how long a finished exchange keeps announcing itself.
// A slow model can take minutes to build a page, so the whole turn counts;
// but come back tomorrow and the canvas is calm.
const turnIsFresh = 2 * time.Minute

// inLastTurn reports whether a moment is one the person is plausibly
// watching: inside the exchange that just finished, or in the last few
// seconds. A glow that outlives its change would be chrome again.
func inLastTurn(ts time.Time, convo *conversation) bool {
	if time.Since(ts) < 10*time.Second {
		return true
	}
	if convo.LastTurn.IsZero() || convo.TurnEnd.Before(convo.LastTurn) {
		return false
	}
	if time.Since(convo.TurnEnd) > turnIsFresh {
		return false
	}
	return !ts.Before(convo.LastTurn) && !ts.After(convo.TurnEnd.Add(2*time.Second))
}

// str reads a string field, falling back when the workspace's schema does
// not define it.
func str(v any, fallback string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return fallback
}

type canvasBlock struct {
	ID         string
	Component  string
	Actor      string
	Changed    string
	Span       int
	Frame      string
	Tone       string
	Provenance string
	HTML       template.HTML
	Remove     template.HTML
}

// canvasDelete is a person removing a block; it is logged as a human action.
func (s *Server) canvasDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rec, err := s.app.Store.Get(chat.BlockType, id)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.app.Store.Delete(chat.BlockType, id); err != nil {
		s.fail(w, err)
		return
	}
	name, _ := rec.Fields["component"].(string)
	props, _ := rec.Fields["props"].(map[string]any)
	chat.Record(s.app.Store, "human", chat.Change{Action: "removed", Component: name, ID: id, Detail: chat.Summarise(name, props)})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
