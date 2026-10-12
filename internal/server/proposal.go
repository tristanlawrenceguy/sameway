package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Proposals are the assistant asking rather than acting. They sit at the end
// of the conversation, where the person is already looking, and nothing
// happens until one of the two answers is given.

// proposals renders every question still waiting for an answer.
func (s *Server) proposals(from string) []template.HTML {
	var out []template.HTML
	for _, p := range s.app.Chat.Proposals() {
		out = append(out, s.proposalCard(p, from))
	}
	return out
}

// proposalCard is one question with its two answers, wherever it is shown:
// under the conversation, or on the proposal's own page. from is the page
// the person is on, so answering brings them back to it.
// cannotUndo are the calls a question can carry that cannot be taken back.
var cannotUndo = map[string]bool{"run_action": true, "accept_action": true, "set_setting": true, "let_in": true, "change_field": true}

func (s *Server) proposalCard(p *store.Record, from string) template.HTML {
	summary, _ := p.Fields["summary"].(string)
	props := map[string]any{"summary": summary, "id": "proposal-" + p.ID}
	// Its answers are the same wherever what waits is read (chat/waiting.go).
	for _, a := range chat.QuestionPresses(p) {
		switch {
		case strings.HasSuffix(a.Post, "/accept"):
			props["accept"], props["acceptLabel"] = a.Post, a.Label
		case strings.HasSuffix(a.Post, "/dismiss"):
			props["dismiss"], props["dismissLabel"] = a.Post, a.Label
		default:
			props["instead"], props["insteadLabel"] = a.Post, a.Label
		}
	}
	if from != "" {
		props["from"] = from
	}
	if detail := proposalDetail(p.Fields["action"]); detail != "" {
		props["detail"] = detail
	}
	// A question Sameway put itself, because what it would do cannot be
	// taken back, says so in its own words.
	if detail, _ := p.Fields["detail"].(string); detail != "" {
		props["detail"] = detail
	}
	// What it would do cannot be taken back: the question says so, and its
	// Yes looks like it.
	if action, _ := p.Fields["action"].(map[string]any); action != nil {
		tool, _ := action["tool"].(string)
		key, _ := action["key"].(string)
		if cannotUndo[tool] && (tool != "set_setting" || chat.Outward(key)) {
			props["risk"] = true
		}
	}
	return s.component("proposal", props)
}

// proposalDetail says exactly what would change, so nobody agrees to a
// surprise on the strength of a summary alone.
func proposalDetail(action any) string {
	a, ok := action.(map[string]any)
	if !ok {
		return ""
	}
	tool, _ := a["tool"].(string)
	component, _ := a["component"].(string)
	switch tool {
	case "add_component":
		if component != "" {
			return "This would add a " + component + " to the page."
		}
	case "remove_component":
		return "This would remove that block from the page."
	case "update_component":
		return "This would change that block."
	case "accept_action":
		return "This runs on your machine, as you, now and every time the button is pressed from now on. Change the command and it asks again."
	}
	return ""
}

func (s *Server) proposalAccept(w http.ResponseWriter, r *http.Request) {
	s.answer(w, r, s.app.Chat.Accept)
}

func (s *Server) proposalDismiss(w http.ResponseWriter, r *http.Request) {
	s.answer(w, r, s.app.Chat.Dismiss)
}

func (s *Server) proposalInstead(w http.ResponseWriter, r *http.Request) {
	s.answer(w, r, s.app.Chat.Instead)
}

// answer applies one answer and reports any problem where the person is
// looking, rather than on an error page they did not ask for.
func (s *Server) answer(w http.ResponseWriter, r *http.Request, apply func(string) error) {
	r.ParseForm()
	if err := apply(r.PathValue("id")); err != nil {
		s.failed(w, r, "That did not work", err, "/")
		return
	}
	// One question is on show at a time, and answering it sets the others
	// aside too, here as on the page, so none is left waiting unseen.
	for _, p := range s.app.Chat.Proposals() {
		s.app.Chat.Dismiss(p.ID)
	}
	http.Redirect(w, r, web.BackOf(r, "/"), http.StatusSeeOther)
}

// questionSection is what the assistant asks, on Today as under the
// conversation: a question asked where the person is not looking, such as
// a command an agent asked to run, waits there too.
func (s *Server) questionSection(list []chat.Waiting) string {
	var b strings.Builder
	for _, w := range chat.WaitingOf(list, chat.WaitQuestion) {
		b.WriteString(string(s.proposalCard(w.Record, "/today")))
	}
	if b.Len() == 0 {
		return ""
	}
	return `<h2>The assistant asks</h2><div class="sw-stack sw-proposals">` + b.String() + `</div>`
}

// proposalsHTML is the questions waiting, as the conversation shows them,
// or nothing when there are none.
func (s *Server) proposalsHTML(from string) string {
	list := s.proposals(from)
	if len(list) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="sw-stack sw-proposals">`)
	for _, p := range list {
		b.WriteString(string(p))
	}
	b.WriteString(`</div>`)
	return b.String()
}
