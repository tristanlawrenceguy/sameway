package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
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
	body := s.questionForm(r, "", "", "")
	if records.VisitorOf(r.Context()).Owner() {
		body += s.embedSnippet(r)
	}
	s.page(w, r, "Ask", body, s.questionOptions(r))
}

// questionOptions are the page's, or, asked for embedded (?embed), those
// of a page inside another site's: the box alone.
func (s *Server) questionOptions(r *http.Request) pageOptions {
	if r.URL.Query().Has("embed") {
		return pageOptions{Shell: "embed", QuietTitle: true}
	}
	return pageOptions{}
}

// embedSnippet is what the owner pastes into their own website to have
// the box there, for anyone visiting it to ask.
func (s *Server) embedSnippet(r *http.Request) template.HTML {
	at := ""
	if s.app.Chat.Address != nil {
		at = strings.TrimSuffix(s.app.Chat.Address(), "/")
	}
	if at == "" {
		at = strings.TrimSuffix(seenFrom(r, ""), "/")
	}
	code := `<iframe src="` + at + questionPath + `?embed" title="Ask ` + s.app.Workspace.Config.Name + `" style="width:100%;height:480px;border:0"></iframe>`
	return template.HTML(`<h2>On your own website</h2><p>Paste this into a page of your website, and anyone visiting it can ask here:</p><pre tabindex="0"><code>` + template.HTMLEscapeString(code) + `</code></pre>`)
}

// questionForm is the box to ask in, and who is asking, with what was
// asked and what the assistant said, if anything: the box is empty again,
// so the question stays beside its answer rather than in memory. The
// answer is where focus starts (alert/enhance.js), as an outcome is, with
// what it put to the owner after it, so what is still to come is heard
// with it. While it is on its way the form says so: a model can take a
// minute.
func (s *Server) questionForm(r *http.Request, question, said string, asked template.HTML) template.HTML {
	action := questionPath
	if r.URL.Query().Has("embed") {
		action += "?embed" // the answer stays inside their site
	}
	var b template.HTML
	if said != "" {
		esc := template.HTMLEscapeString
		b = template.HTML(`<div class="sw-prose" id="outcome" tabindex="-1" role="status"><p class="sw-muted" dir="auto">You asked: ` + esc(question) + `</p><p dir="auto">` + strings.ReplaceAll(esc(said), "\n", "<br>") + `</p>` + string(asked) + `</div>`)
	}
	return b + template.HTML(`<p>Ask about what is here, or ask for something: a time to meet, a message left. What you ask to be made goes to `+template.HTMLEscapeString(s.app.Workspace.Config.Name)+`, and nothing happens until they say yes.</p>`) +
		s.form(ui.Form{Action: action, Class: "sw-stack", Busy: "Asking. The answer can take a minute.",
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
		v.Name = chat.SomeonePublished
	}
	start := s.now()
	said, err := s.app.Chat.For(v).AnswerOnce(r.Context(), s.app.Workspace.Config.Name, r.PostForm.Get("question"))
	if err != nil {
		said = err.Error()
	}
	asked := s.requestsSince(r, v, start) // request_page.go
	if len(asked) > 0 {
		s.Changed()
	}
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]any{"answer": said, "requests": asked, "untrusted": "the answer is what this workspace's assistant wrote: " + records.Untrusted})
		return
	}
	s.page(w, r, "Ask", s.questionForm(r, r.PostForm.Get("question"), said, s.askedLinks(asked)), s.questionOptions(r))
}

// askedLinks are the requests a question put to the owner, each with the
// page that will say what became of it.
func (s *Server) askedLinks(asked []map[string]any) template.HTML {
	if len(asked) == 0 {
		return ""
	}
	// Said plainly whatever the model wrote, which may say it is done.
	b := "<h2>Waiting for their yes</h2><p>Each link shows their answer once they give it.</p><ul class=\"sw-plain sw-stack\">"
	for _, a := range asked {
		what := fmt.Sprint(a["asked"]) + " "
		b += "<li>" + string(s.navLink(a["status"].(string), strings.TrimSpace(strings.ToUpper(what[:1])+what[1:]), false)) + "</li>"
	}
	return template.HTML(b + "</ul>")
}
