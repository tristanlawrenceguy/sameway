package server

import (
	"fmt"
	"html"
	"html/template"
	"net"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
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
	body := s.questionForm(r, asking{})
	if records.VisitorOf(r.Context()).Owner() && !r.URL.Query().Has("embed") {
		body += s.embedSnippet(r)
	}
	s.page(w, r, "Ask", body, s.questionOptions(r, asking{}))
}

// asking is one question asked on the page and what came of it: said
// back with the answer, so nobody has to remember what they asked, and
// who they are kept for the next one.
type asking struct {
	question, who, said, problem string
}

// questionOptions are the page's, or, asked for embedded (?embed), those
// of a page inside another site's: the box alone. After an answer, the
// first skip link goes to it.
func (s *Server) questionOptions(r *http.Request, a asking) pageOptions {
	var o pageOptions
	if r.URL.Query().Has("embed") {
		o = pageOptions{Shell: "embed", QuietTitle: true}
	}
	switch {
	case a.problem != "":
		o.Focus, o.FocusLabel = "question", "Skip to what to fix"
	case a.said != "":
		o.Focus, o.FocusLabel = "answer", "Skip to the answer"
	}
	return o
}

// embedSnippet is what the owner pastes into their own website to have
// the box there, for anyone visiting it to ask. It is wrapped, not
// scrolled, so it reads whole on a phone and nothing needs a scroll key.
func (s *Server) embedSnippet(r *http.Request) template.HTML {
	at := s.internetAt(r)
	if at == "" {
		return `<h2>On your own website</h2><p>Once the internet reaches this workspace, what to paste into your website is here.</p>`
	}
	code := `<iframe src="` + html.EscapeString(at+questionPath+"?embed") + `" title="` + html.EscapeString("Ask "+s.app.Workspace.Config.Name) + `" style="width:100%;height:480px;border:0"></iframe>`
	return template.HTML(`<h2>On your own website</h2><p>Paste this into a page of your website, and anyone visiting it can ask here:</p><p><code>` + template.HTMLEscapeString(code) + `</code></p>`)
}

// internetAt is where the internet reaches this workspace, or "" when it
// cannot yet: an address given on localhost would work nowhere else.
func (s *Server) internetAt(r *http.Request) string {
	if s.app.Chat.Address != nil {
		if at := strings.TrimSuffix(s.app.Chat.Address(), "/"); at != "" {
			return at
		}
	}
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host == "" || host == "localhost" || net.ParseIP(strings.Trim(host, "[]")) != nil {
		return ""
	}
	return strings.TrimSuffix(seenFrom(r, ""), "/")
}

// questionForm is the box to ask in, and who is asking, with what they
// asked and what the assistant said, if anything.
func (s *Server) questionForm(r *http.Request, a asking) template.HTML {
	esc := template.HTMLEscapeString
	action := questionPath
	if r.URL.Query().Has("embed") {
		action += "?embed" // the answer stays inside their site
	}
	var b template.HTML
	if a.said != "" {
		b = template.HTML(`<div class="sw-prose" id="answer" role="status"><p>You asked: <span dir="auto">` + esc(a.question) + `</span></p><p dir="auto">` + strings.ReplaceAll(esc(a.said), "\n", "<br>") + `</p></div>`)
	}
	return b + template.HTML(`<p>Ask about what is here, or ask for something: a time to meet, a message left. What you ask to be made goes to `+esc(s.app.Workspace.Config.Name)+`, and nothing happens until they say yes.</p>`) +
		s.form(ui.Form{Action: action, Class: "sw-stack",
			Body: s.component("textarea", map[string]any{"label": "Your question", "name": "question", "rows": 3, "maxlength": 1000, "required": true, "error": a.problem}) +
				s.part(ui.TextField{Label: "Your name and how to reach you", Name: "who", Value: a.who, MaxLength: 160, Hint: "Optional; needed for anything you ask to be made.", Autocomplete: "name"}),
			Button: &ui.Button{Label: "Ask", Variant: ui.Primary}})
}

func (s *Server) questionAsk(w http.ResponseWriter, r *http.Request) {
	if !s.Published().Any() {
		http.NotFound(w, r)
		return
	}
	r.ParseForm()
	a := asking{question: strings.TrimSpace(r.PostForm.Get("question")), who: strings.TrimSpace(r.PostForm.Get("who"))}
	v := records.Visitor{Access: records.Public, Name: a.who}
	if a.who == "" {
		v.Name = chat.SomeonePublished
	}
	said, made, err := s.app.Chat.For(v).AnswerOnce(r.Context(), s.app.Workspace.Config.Name, a.question)
	if err != nil {
		said = web.PlainError(err)
	}
	if a.question == "" {
		a.problem = said // the field to fix says so, with aria-invalid
	} else {
		a.said = said
	}
	asked := s.requestsSeen(r, made) // request_page.go
	if len(asked) > 0 {
		s.Changed()
	}
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"answer": said, "requests": asked, "untrusted": "the answer is what this workspace's assistant wrote: " + records.Untrusted})
		return
	}
	s.page(w, r, "Ask", s.questionForm(r, a)+s.askedLinks(asked), s.questionOptions(r, a))
}

// askedLinks are the requests a question put to the owner, each with the
// page that will say what became of it.
func (s *Server) askedLinks(asked []map[string]any) template.HTML {
	if len(asked) == 0 {
		return ""
	}
	b := "<h2>What you asked for</h2><ul class=\"sw-plain sw-stack\">"
	for _, a := range asked {
		b += "<li>" + string(s.navLink(a["status"].(string), "To "+fmt.Sprint(a["asked"])+": see the answer", false)) + "</li>"
	}
	return template.HTML(b + "</ul>")
}
