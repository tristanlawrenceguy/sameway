package server

import (
	"fmt"
	"html/template"
	"net/url"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// A new workspace opened on an empty page with "Ask for anything" and four
// things to try, and a person had to know already what Sameway is for and
// how they would use it. A workspace with nothing in it yet now opens on a
// welcome: Sameway is set up around what they keep track of, and four
// common starting points each put a sentence in the box that asks the
// assistant to do it, which they send or change first. The assistant asks
// at most two short questions and builds it (the prompt's set-up line).

// setups are the starting points: what the button says, and what it asks.
var setups = []struct{ label, ask string }{
	{"Work: tasks, meetings, clients", "Set Sameway up for my work. I want to keep track of tasks, meetings and clients."},
	{"Home: shopping, chores, bills", "Set Sameway up for my home. I want to keep track of shopping, chores and bills."},
	{"Study: courses, deadlines, notes", "Set Sameway up for my studies. I want to keep track of my courses, deadlines and notes."},
	{"Health: habits, appointments", "Set Sameway up for my health. I want to keep track of habits and appointments."},
}

// brandNew says whether the workspace has nothing of the person's in it
// yet: no record of any kind they keep, and no conversation.
func (s *Server) brandNew() bool {
	for _, t := range s.app.Types.Types {
		if t.Internal {
			continue
		}
		if n, _ := s.app.Store.Count(t.Name); n > 0 {
			return false
		}
	}
	n, _ := s.app.Store.Count(chat.MessageType)
	return n == 0
}

// welcome is what an empty conversation in a brand-new workspace shows.
func (s *Server) welcome(from string) template.HTML {
	esc := template.HTMLEscapeString
	b := string(s.component("empty", map[string]any{"message": "Welcome to Sameway. Tell the assistant what you want to keep track of and it sets Sameway up around it. Start from one of these, or say it in your own words."})) +
		`<ul class="sw-plain sw-chat__starts" aria-label="Ways to start">`
	for _, st := range setups {
		b += fmt.Sprintf(`<li><a class="sw-link sw-link--button sw-chat__start" href="%s?prompt=%s">%s</a></li>`, esc(from), url.QueryEscape(st.ask), esc(st.label))
	}
	b += `</ul><p class="sw-small">Or start from a <a class="sw-link" href="/templates">template</a>: a job search, clients, a budget, a house move and more.</p>`
	return template.HTML(b)
}
