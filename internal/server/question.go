package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// A visitor from the internet asks the assistant on a published
// workspace: /question, while anything is published. It works for them
// with their reach, and no more (chat/answer_once.go): it reads what is
// published, and what they ask to be made is a request on the owner's
// Today, made only with the owner's yes. Nothing else they write is kept.

const (
	questionPath = "/question"
	questionGet  = "GET /question"
	questionPost = "POST /question"
)

func (s *Server) questionPage(w http.ResponseWriter, r *http.Request) {
	if !s.Published().Any() {
		http.NotFound(w, r)
		return
	}
	s.page(w, r, "Ask", s.questionForm(""), pageOptions{})
}

// questionForm is the box to ask in, and who is asking, with what the
// assistant said last, if anything.
func (s *Server) questionForm(said string) template.HTML {
	var b template.HTML
	if said != "" {
		b = template.HTML(`<div class="sw-prose" role="status"><p>` + strings.ReplaceAll(template.HTMLEscapeString(said), "\n", "<br>") + `</p></div>`)
	}
	return b + template.HTML(`<p>Ask about what is here, or ask for something: a time to meet, a message left. What you ask to be made goes to `+template.HTMLEscapeString(s.app.Workspace.Config.Name)+`, and nothing happens until they say yes.</p>`) +
		s.form(ui.Form{Action: questionPath, Class: "sw-stack",
			Body: s.component("textarea", map[string]any{"label": "Your question", "name": "question", "rows": 3, "maxlength": 1000}) +
				s.part(ui.TextField{Label: "Your name and how to reach you", Name: "who", MaxLength: 160, Hint: "Optional; needed for anything you ask to be made.", Autocomplete: "name"}),
			Button: &ui.Button{Label: "Ask", Variant: ui.Primary}})
}

func (s *Server) questionAsk(w http.ResponseWriter, r *http.Request) {
	if !s.Published().Any() {
		http.NotFound(w, r)
		return
	}
	r.ParseForm()
	who := strings.TrimSpace(r.PostForm.Get("who"))
	v := records.Visitor{Access: records.Public, Name: who}
	if who == "" {
		v.Name = "Someone on your published page"
	}
	said, err := s.app.Chat.For(v).AnswerOnce(r.Context(), s.app.Workspace.Config.Name, r.PostForm.Get("question"))
	if err != nil {
		said = err.Error()
	}
	if len(s.app.Chat.Suggestions()) > 0 {
		s.Changed()
	}
	s.page(w, r, "Ask", s.questionForm(said), pageOptions{})
}
