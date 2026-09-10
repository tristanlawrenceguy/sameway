package server

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Proposals are the assistant asking rather than acting. They sit at the end
// of the conversation, where the person is already looking, and nothing
// happens until one of the two answers is given.

// proposals renders every question still waiting for an answer.
func (s *Server) proposals() []template.HTML {
	var out []template.HTML
	for _, p := range s.app.Chat.Proposals() {
		summary, _ := p.Fields["summary"].(string)
		props := map[string]any{
			"summary": summary,
			"accept":  "/proposal/" + p.ID + "/accept",
			"dismiss": "/proposal/" + p.ID + "/dismiss",
			"id":      "proposal-" + p.ID,
		}
		if detail := proposalDetail(p.Fields["action"]); detail != "" {
			props["detail"] = detail
		}
		if label := acceptLabel(p.Fields["action"]); label != "" {
			props["acceptLabel"] = label
		}
		out = append(out, s.component("proposal", props))
	}
	return out
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
	}
	return ""
}

func acceptLabel(action any) string {
	a, ok := action.(map[string]any)
	if !ok {
		return ""
	}
	switch tool, _ := a["tool"].(string); tool {
	case "add_component":
		return "Yes, add it"
	case "remove_component":
		return "Yes, remove it"
	case "update_component":
		return "Yes, change it"
	}
	return ""
}

func (s *Server) proposalAccept(w http.ResponseWriter, r *http.Request) {
	s.answer(w, r, s.app.Chat.Accept)
}

func (s *Server) proposalDismiss(w http.ResponseWriter, r *http.Request) {
	s.answer(w, r, s.app.Chat.Dismiss)
}

// answer applies one answer and reports any problem where the person is
// looking, rather than on an error page they did not ask for.
func (s *Server) answer(w http.ResponseWriter, r *http.Request, apply func(string) error) {
	if err := apply(r.PathValue("id")); err != nil {
		s.app.Store.Create(chat.MessageType, map[string]any{
			"role": "error", "content": "That did not go through. " + err.Error(),
		})
	}
	r.ParseForm()
	back := "/"
	if from := r.PostForm.Get("from"); strings.HasPrefix(from, "/") {
		back = from
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}
