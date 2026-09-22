package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Popping a block out: the same block, given the whole middle of the page.
//
// Nothing is moved or copied to do it. The block stays exactly where it is
// on the canvas; this is another way of looking at it, at its own URL, so a
// person can send it to someone and an agent can ask for it directly. A
// component that has more to show at that size is asked for its fullest
// form, and one that has not simply arrives larger.
func (s *Server) focusPage(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(chat.BlockType, r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	name, _ := rec.Fields["component"].(string)
	comp, ok := s.app.Registry.Get(name)
	if !ok {
		s.fail(w, err)
		return
	}
	convo, err := s.conversation("/canvas/" + rec.ID)
	if err != nil {
		s.fail(w, err)
		return
	}

	convo.FocusID = rec.ID

	props, _ := rec.Fields["props"].(map[string]any)
	if comp.Manifest.Name == recordComponent {
		// The record's own words, so the page is named after them and the
		// expanded block shows them.
		props, _ = s.resolveRecord(props)
	}
	if comp.Manifest.Name == collectionComponent {
		props = s.resolveCollection(props)
	}
	if comp.Manifest.Name == calendarComponent {
		// The block's own page takes ?month= and ?day= so the months and
		// days either side are a link away, and the calendar comes back to
		// this page for them.
		if m, d := r.URL.Query().Get("month"), r.URL.Query().Get("day"); m != "" || d != "" {
			props = withMonth(props, m, d)
		}
		props = s.resolveCalendar(props, rec.ID)
	}
	if comp.Manifest.Name == clockComponent {
		props = s.resolveClock(props)
	}
	if comp.Manifest.Name == trackerComponent {
		props = s.resolveTracker(props)
	}
	if comp.Manifest.Name == chartComponent {
		props = s.resolveChart(props)
	}
	body := s.expanded(comp.Manifest.Name, props, convo)

	var b strings.Builder
	b.WriteString(`<div class="sw-focus">`)
	b.WriteString(string(s.component("link", map[string]any{"href": chat.CanvasPath(canvasOf(rec.Fields)), "label": "Back to the canvas", "look": "button"})))
	// The block's own page wears its list's colour, as the block does on
	// the canvas.
	dot := ""
	if typeName, _ := props["type"].(string); typeName != "" {
		if n := s.dotOf(typeName); n > 0 {
			dot = fmt.Sprintf(` data-dot="%d"`, n)
		}
	}
	b.WriteString(`<div class="sw-focus__body"` + dot + `>` + string(body) + `</div></div>`)

	reg := split(s.canvasBlocks())
	left, right := reg.left, reg.right
	name, own := title(comp.Manifest.Name, props)
	s.page(w, r, name, template.HTML(b.String()), pageOptions{
		// When the block already says what it is (a calendar's caption, a
		// card's title), the page heading would say it a second time. It
		// stays in the outline for anyone navigating by headings, and off
		// the screen for everyone else.
		QuietTitle: own,
		JSONURL:    "/api/block/" + rec.ID,
		Left:       s.pane("left", "History", left, convo),
		Right:      s.pane("right", "Alongside", right, convo),
	})
}

// expanded renders a component at its fullest, when it has one.
func (s *Server) expanded(name string, props map[string]any, convo *conversation) template.HTML {
	if name == chat.ComponentName {
		out, err := s.app.Registry.RenderSlot(name, props, convo.Body)
		if err == nil {
			return out
		}
	}
	comp, ok := s.app.Registry.Get(name)
	if !ok {
		return s.component(name, props)
	}
	full := map[string]any{}
	for k, v := range props {
		full[k] = v
	}
	if comp.HasProp("detail") {
		full["detail"] = "page"
	}
	if comp.HasProp("level") {
		full["level"] = int64(2)
	}
	if out, err := comp.Render(full); err == nil {
		return out
	}
	return s.component(name, props)
}

// title names the expanded page after what the block actually says, so the
// browser tab and the heading are about the thing, not about the software.
// The second return says whether the name came from the block's own words,
// which is also whether the block is already showing them.
func title(name string, props map[string]any) (string, bool) {
	for _, key := range []string{"caption", "title", "label", "text"} {
		if v, ok := props[key].(string); ok && strings.TrimSpace(v) != "" {
			return truncateTitle(v), true
		}
	}
	return strings.ToUpper(name[:1]) + name[1:], false
}

func truncateTitle(s string) string {
	s = strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	if len([]rune(s)) <= 60 {
		return s
	}
	return string([]rune(s)[:59]) + "…"
}
