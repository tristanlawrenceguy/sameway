package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Templates: ready-made pages for a job (a job search, clients, a house
// move, a budget, recipes, studies, health, a trip, reading, writing, the
// week), each a press away (chat.UseTemplate): the kinds of record it
// needs are made, a tab of its own holds it, and it shows the person's own
// records from then on, empty until they add some. Linked from the new
// workspace's welcome and from Help. Why it reads as it does is in
// design/foundations/glance.md, "Templates".

func (s *Server) templatesPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	b.WriteString(`<p>Each makes a tab of its own, with the lists and calendars that job needs. They show your own things, empty until you add some, by asking or by hand. One Undo takes a tab back.</p><ul class="sw-plain sw-stack">`)
	for _, t := range s.app.Chat.Templates() {
		b.WriteString(s.templateItem(t))
	}
	b.WriteString(`</ul>`)
	s.page(w, r, "Templates", template.HTML(b.String()), pageOptions{Lede: "Ready-made pages for a job, each a press away."})
}

// templateItem is one template: what it is for, what its tab shows and
// adds, in words, then the one press.
func (s *Server) templateItem(t chat.Template) string {
	esc := template.HTMLEscapeString
	out := `<li class="sw-stack"><h2>` + esc(t.Title) + `</h2><p>` + esc(t.Description) + `</p>`
	if len(t.Shows) > 0 {
		out += `<p class="sw-small">Its tab shows: ` + esc(blocks.AndList(t.Shows)) + `.</p>`
	}
	if len(t.Makes) > 0 {
		out += `<p class="sw-small sw-muted">Adds a kind of thing to keep: ` + esc(plurals(t.Makes)) + `.</p>`
	}
	if t.Have != "" {
		out += `<p class="sw-small">You have a <a class="sw-link" href="` + esc(records.CanvasPath(t.Have)) + `">` + esc(t.Title) + `</a> tab already. Using it again makes another.</p>`
	}
	return out + string(s.form(ui.Form{Action: "/templates/use", Hidden: ui.Hidden("name", t.Name), Button: &ui.Button{Label: "Use this template", Context: t.Title, Variant: ui.Secondary}})) + `</li>`
}

func plurals(kinds []string) string {
	var out []string
	for _, k := range kinds {
		out = append(out, schema.Plural(k))
	}
	return strings.Join(out, ", ")
}

func (s *Server) templateUse(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	used, err := s.chatFor(r).UseTemplate(r.PostForm.Get("name"), "human")
	if err != nil {
		s.failed(w, r, "Not made", err, "/templates")
		return
	}
	text := used.Name + " is a tab of its own now."
	if len(used.Makes) > 0 {
		text += " It keeps " + plurals(used.Makes) + "."
	}
	text += " Add your own things here, or ask the assistant to."
	s.tellAt(w, r, outcome{Title: "Ready", Text: text, Undo: used.Activity, Of: "the " + used.Name + " tab"}, records.CanvasPath(used.Tab))
}
