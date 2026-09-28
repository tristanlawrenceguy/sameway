package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// act runs one of the person's actions from a button and returns them to
// the page they pressed it on. What happened is in the activity log, where
// every other change is; a failure is said there too, rather than on an
// error page they did not ask for. A command not yet accepted is a
// question: the person is taken to it, with the command line in front of
// them and the two answers.
func (s *Server) act(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	back := backOf(r, "/")
	// The tab the button was on is where a message action's reply lands.
	canvas := strings.TrimPrefix(back, "/c/")
	if !strings.HasPrefix(back, "/c/") {
		canvas = ""
	}
	text, proposal, err := s.app.Chat.RunAs(r.Context(), "human", r.PathValue("id"), canvas)
	if err != nil {
		s.failed(w, r, "Did not run", err, "/")
		return
	}
	if proposal != "" {
		http.Redirect(w, r, "/t/"+chat.ProposalType+"/"+proposal, http.StatusSeeOther)
		return
	}
	// What it came back with, in a sentence or two.
	if r := []rune(strings.TrimSpace(text)); len(r) > 240 {
		text = string(r[:239]) + "…"
	}
	s.tellAt(w, r, outcome{Title: "Done", Text: strings.TrimSpace(text)}, back)
}

// apiAct runs an action for an agent and answers with what came back, or
// with where the person's acceptance is waiting.
func (s *Server) apiAct(w http.ResponseWriter, r *http.Request) {
	text, proposal, err := s.app.Chat.RunAs(r.Context(), "human", r.PathValue("id"), "")
	if err != nil {
		writeError(w, errors.New(err.Error()))
		return
	}
	out := map[string]any{"ran": r.PathValue("id"), "result": text}
	if proposal != "" {
		out["waiting_for"] = "/t/" + chat.ProposalType + "/" + proposal
	}
	writeJSON(w, http.StatusOK, out)
}

// hook lets something outside press a button: a request to /hook/<word>
// runs the action whose trigger is that word, as the system, and answers
// with what came back. An action without a trigger cannot be reached this
// way, and a command not yet accepted stays a question for the person.
func (s *Server) hook(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	recs, _ := s.app.Store.List(chat.ActionType, store.ListOptions{})
	for _, rec := range recs {
		if t, _ := rec.Fields["trigger"].(string); t == "" || t != token {
			continue
		}
		text, proposal, err := s.app.Chat.RunAs(r.Context(), "system", rec.ID, "")
		if err != nil {
			writeError(w, errors.New(err.Error()))
			return
		}
		out := map[string]any{"ran": rec.ID, "result": text}
		if proposal != "" {
			out["waiting_for"] = "/t/" + chat.ProposalType + "/" + proposal
		}
		writeJSON(w, http.StatusOK, out)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]any{"error": apiError{Code: "not_found", Message: "no action has that trigger"}})
}
