package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// act runs one of the person's actions from a button and returns them to
// the page they pressed it on. What happened is in the activity log, where
// every other change is; a failure is said there too, rather than on an
// error page they did not ask for.
func (s *Server) act(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	back := "/"
	if from := r.PostForm.Get("from"); strings.HasPrefix(from, "/") && !strings.HasPrefix(from, "//") {
		back = from
	}
	// The tab the button was on is where a message action's reply lands.
	canvas := strings.TrimPrefix(back, "/c/")
	if !strings.HasPrefix(back, "/c/") {
		canvas = ""
	}
	if _, err := s.app.Chat.RunAs(r.Context(), "human", r.PathValue("id"), canvas); err != nil {
		chat.Record(s.app.Store, "system", chat.Change{Action: "failed", Detail: "action: " + err.Error()})
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// apiAct runs an action for an agent and answers with what came back.
func (s *Server) apiAct(w http.ResponseWriter, r *http.Request) {
	text, err := s.app.Chat.RunAs(r.Context(), "human", r.PathValue("id"), "")
	if err != nil {
		writeError(w, errors.New(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ran": r.PathValue("id"), "result": text})
}
