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
	if len(blocks) == 0 {
		b.WriteString(`<p class="sw-empty">This canvas is empty. The conversation is at <a href="/chat">/chat</a>, and anything you ask for there appears here.</p>`)
	} else {
		b.WriteString(`<ol class="sw-plain sw-canvas" aria-label="Canvas">`)
		for _, blk := range blocks {
			b.WriteString(s.blockItem(blk, convo))
		}
		b.WriteString(`</ol>`)
	}
	if convo.Activity != "" {
		b.WriteString(`<div class="sw-activity">` + string(convo.Activity) + `</div>`)
	}
	opts := pageOptions{JSONURL: "/api/block"}
	if convo.LatestID != "" {
		opts.Focus, opts.FocusLabel = convo.LatestID, "Skip to latest message"
	}
	s.page(w, r, "Canvas", template.HTML(b.String()), opts)
}

// blockItem renders one canvas block: the component, its span, its
// provenance rail, and its quiet control bar.
func (s *Server) blockItem(blk *store.Record, convo *conversation) string {
	v := s.canvasBlock(blk, convo.LastTurn)
	body := v.HTML
	if v.Component == chat.ComponentName {
		body = s.chatBlock(blk, convo)
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<li class="sw-block sw-reveal" data-block-id="%s" data-block-component="%s" data-actor="%s"`, v.ID, v.Component, v.Actor)
	if v.Changed != "" {
		fmt.Fprintf(&b, ` data-changed="%s"`, v.Changed)
	}
	fmt.Fprintf(&b, ` style="--sw-span: %d; view-transition-name: block-%s; view-transition-class: sw-vt-item">`, v.Span, v.ID)
	b.WriteString(string(body))
	fmt.Fprintf(&b, `<div class="sw-bar sw-quiet">%s%s<form method="post" action="/canvas/%s/delete">%s</form></div></li>`,
		v.Badge, v.Edit, v.ID, v.Remove)
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
func (s *Server) canvasBlock(b *store.Record, lastTurn time.Time) canvasBlock {
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
	// Say only what is not obvious. Who made it is the whole fact when one
	// actor did everything; the second clause appears only when the other
	// one has been in since. Times live in the activity log.
	who := map[string]string{"human": "You", "assistant": "Assistant"}
	label := who[createdBy]
	if actor != createdBy {
		label += ", edited by " + strings.ToLower(who[actor])
	}

	changed := ""
	if !lastTurn.IsZero() {
		if !b.CreatedAt.Before(lastTurn) {
			changed = "added"
		} else if !b.UpdatedAt.Before(lastTurn) {
			changed = "updated"
		}
	}
	// Labels stay short on screen and carry the component name in the
	// accessible name, so "Edit" reads as "Edit card" to a screen reader or
	// an agent targeting by role and name.
	return canvasBlock{
		ID: b.ID, Component: name, Actor: actor, Changed: changed, Span: span,
		HTML:   s.component(name, props),
		Badge:  s.component("badge", map[string]any{"label": label, "tone": actor}),
		Edit:   s.component("link", map[string]any{"href": "/t/block/" + b.ID + "/edit", "label": "Edit", "context": name}),
		Remove: s.component("button", map[string]any{"label": "Remove", "context": name, "type": "submit", "variant": "quiet"}),
	}
}

type canvasBlock struct {
	ID        string
	Component string
	Actor     string
	Changed   string
	Span      int
	HTML      template.HTML
	Badge     template.HTML
	Edit      template.HTML
	Remove    template.HTML
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
