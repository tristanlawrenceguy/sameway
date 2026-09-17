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
	// Which tab: Home at /, or a canvas record at /c/<id>.
	canvas := r.PathValue("canvas")
	if !s.app.Chat.HasCanvas(canvas) {
		http.NotFound(w, r)
		return
	}
	s.seedChat(canvas)
	blocks, err := s.app.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
	if err != nil {
		s.fail(w, err)
		return
	}
	blocks = chat.OnCanvas(blocks, canvas)
	convo, err := s.conversation(chat.CanvasPath(canvas))
	if err != nil {
		s.fail(w, err)
		return
	}
	convo.Arrival = arrivals(blocks, convo)

	var b strings.Builder
	if convo.Notice != "" {
		b.WriteString(string(convo.Notice))
	}
	b.WriteString(string(s.tabBar(canvas)))
	reg := split(blocks)
	main, left, right := reg.main, reg.left, reg.right
	// A workspace with nothing but the conversation shows just that, in the
	// middle of the page, the way every other assistant opens. Everything
	// else arrives because someone asked for it.
	solo := len(left) == 0 && len(right) == 0 && len(main) == 1 &&
		main[0].Fields["component"] == chat.ComponentName

	if len(blocks) == 0 {
		b.WriteString(`<p class="sw-empty">Nothing here yet. Ask for something in the <a href="/chat">chat</a> and it appears here.</p>`)
	} else {
		fmt.Fprintf(&b, `<div class="sw-page" data-layout="%s">`, layoutName(solo))
		b.WriteString(`<ol class="sw-plain sw-canvas" aria-label="Canvas">`)
		for _, blk := range main {
			b.WriteString(s.blockItem(blk, convo))
		}
		b.WriteString(`</ol></div>`)
	}
	if convo.Activity != "" {
		b.WriteString(`<div class="sw-activity">` + string(convo.Activity) + `</div>`)
	}
	opts := pageOptions{JSONURL: "/api/block"}
	if convo.LatestID != "" {
		opts.Focus, opts.FocusLabel = convo.LatestID, "Skip to latest message"
	}
	// The canvas is an application whether or not it has panes, so it keeps
	// the whole width and the same shape as panes come and go. Once it holds
	// anything, what is on it is the title; "Canvas" stays in the outline.
	opts.Shell = "app"
	opts.QuietTitle = len(blocks) > 0
	opts.Left = s.pane("left", paneLabel("Left pane", left), left, convo)
	opts.Right = s.pane("right", paneLabel("Right pane", right), right, convo)
	opts.Header = s.strip("header", reg.header, convo)
	opts.Footer = s.strip("footer", reg.footer, convo)
	s.page(w, r, "Canvas", template.HTML(b.String()), opts)
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
	fmt.Fprintf(&b, `<li class="sw-block sw-reveal" data-block-id="%s" data-block-component="%s" data-actor="%s" data-frame="%s" data-tone="%s" data-size="%s"`,
		v.ID, v.Component, v.Actor, v.Frame, v.Tone, v.Size)
	if v.Changed != "" {
		fmt.Fprintf(&b, ` data-changed="%s"`, v.Changed)
	}
	if n := convo.Arrival[v.ID]; n > 0 {
		fmt.Fprintf(&b, ` data-arrival="%d"`, n)
	}
	if v.EditAction != "" {
		fmt.Fprintf(&b, ` data-edit-action="%s"`, v.EditAction)
	}
	fmt.Fprintf(&b, ` style="--sw-span: %d; view-transition-name: block-%s; view-transition-class: sw-vt-item">`, v.Span, v.ID)
	// At icon size the block is a glyph with its name, opening the whole
	// thing on its own page: everything is still reachable, in less room.
	if v.Size == "icon" {
		fmt.Fprintf(&b, `<a class="sw-block__icon" href="/canvas/%s" aria-label="%s"><span aria-hidden="true">%s</span></a></li>`, v.ID, template.HTMLEscapeString(v.Label), template.HTMLEscapeString(v.Icon))
		return b.String()
	}
	// Provenance costs nothing on screen and is complete in the
	// accessibility tree. Sighted people got it from the glow when it
	// happened, and can get it again from the activity log.
	fmt.Fprintf(&b, `<p class="sw-visually-hidden">%s</p>`, template.HTMLEscapeString(v.Provenance))
	b.WriteString(string(body))
	fmt.Fprintf(&b, `<div class="sw-bar sw-quiet">%s<form method="post" action="/canvas/%s/delete">%s</form></div></li>`,
		v.Expand, v.ID, v.Remove)
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
	who := map[string]string{"human": "you", "assistant": "the assistant", "system": "the workspace"}
	provenance := "Added by " + who[createdBy] + "."
	if actor != createdBy {
		provenance = "Added by " + who[createdBy] + ", edited by " + who[actor] + "."
	}

	// Glow only for what changed in the exchange just finished, or in the
	// last few seconds, so a marker never outlives the change it reports.
	changed := ""
	// What the workspace put there to begin with is not a change.
	if b.Fields["created_by"] == "system" {
	} else if inLastTurn(b.CreatedAt, convo) {
		changed = "added"
	} else if inLastTurn(b.UpdatedAt, convo) {
		changed = "updated"
	}
	// Removing is the one thing worth a control of its own. Anything else a
	// person wants changed, they ask for, which is faster than any form and
	// is the whole point of having an assistant on the page.
	editAction := ""
	if name == recordComponent {
		props, editAction = s.resolveRecord(props)
	}
	if name == collectionComponent {
		props = s.resolveCollection(props)
	}
	if name == calendarComponent {
		props = s.resolveCalendar(props)
	}
	if name == chartComponent {
		props = s.resolveChart(props)
	}
	label := chat.Summarise(name, props)
	if label == "" {
		label = name
	}
	icon := name[:1]
	if c, ok := s.app.Registry.Get(name); ok && c.Manifest.Icon != "" {
		icon = c.Manifest.Icon
	}
	return canvasBlock{
		ID: b.ID, Component: name, Actor: actor, Changed: changed, Span: span,
		Frame: str(b.Fields["frame"], "card"), Tone: str(b.Fields["tone"], "none"),
		Size: str(b.Fields["size"], "full"), Label: label, Icon: icon,
		Provenance: provenance, EditAction: editAction,
		HTML: s.component(name, props),
		Expand: s.component("link", map[string]any{
			"href": "/canvas/" + b.ID, "label": "Expand", "context": name,
			"current": convo != nil && convo.FocusID == b.ID,
		}),
		Remove: s.component("button", map[string]any{"label": "Remove", "context": name, "type": "submit", "variant": "quiet"}),
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
	ID        string
	Component string
	Actor     string
	Changed   string
	Span      int
	Frame     string
	Tone      string
	// Size is full, compact or icon; Label and Icon are what an icon-sized
	// block shows: the glyph, and the name assistive technology gets.
	Size, Label, Icon string
	Provenance        string
	HTML              template.HTML
	Expand            template.HTML
	Remove            template.HTML
	// EditAction is where the inline editor posts for this block when it is
	// not the block's own props: a record block edits the record.
	EditAction string
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
	chat.Record(s.app.Store, "human", chat.Change{Action: "removed", Component: name, ID: id, Detail: chat.Summarise(name, props), Before: rec.Fields})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
