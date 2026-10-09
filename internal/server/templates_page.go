package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// Templates: ready-made pages for a job (a job search, clients, a house
// move, a budget, recipes, studies, health, a trip, reading, writing, the
// week), each a press away (chat.UseTemplate): the kinds of record it
// needs are made, a tab of its own holds it, and it shows the person's own
// records from then on, empty until they add some. Linked from the new
// workspace's welcome and from Help.

func (s *Server) templatesPage(w http.ResponseWriter, r *http.Request) {
	esc := template.HTMLEscapeString
	var b strings.Builder
	b.WriteString(`<p>Each makes a tab of its own, with the lists and calendars that job needs. They show your own things, empty until you add some, by asking or by hand.</p><ul class="sw-plain sw-stack">`)
	for _, t := range s.app.Chat.Templates() {
		b.WriteString(`<li class="sw-stack"><h2>` + esc(t.Title) + `</h2><p>` + esc(t.Description) + `</p>`)
		if len(t.Makes) > 0 {
			var kinds []string
			for _, k := range t.Makes {
				kinds = append(kinds, schema.Plural(k))
			}
			b.WriteString(`<p class="sw-small sw-muted">Adds a kind of thing to keep: ` + esc(strings.Join(kinds, ", ")) + `.</p>`)
		}
		b.WriteString(string(s.form(ui.Form{Action: "/templates/use", Hidden: ui.Hidden("name", t.Name), Button: &ui.Button{Label: "Use this template", Context: t.Title, Variant: ui.Secondary}})) + `</li>`)
	}
	b.WriteString(`</ul>`)
	s.page(w, r, "Templates", template.HTML(b.String()), pageOptions{Lede: "Ready-made pages for a job, each a press away."})
}

func (s *Server) templateUse(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	id, err := s.chatFor(r).UseTemplate(r.PostForm.Get("name"), "human")
	if err != nil {
		s.failed(w, r, "Not made", err, "/templates")
		return
	}
	s.tellAt(w, r, outcome{Title: "Ready", Text: "The template is a tab of its own now. Add what you keep there, or ask the assistant to."}, records.CanvasPath(id))
}
