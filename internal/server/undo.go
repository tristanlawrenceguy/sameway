package server

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// undo reverses one activity entry for the person and returns them to the
// page they pressed Undo on. A reversal that cannot be done is said in the
// log, where they are looking, rather than on an error page.
func (s *Server) undo(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	// What is undone, said with the outcome: "Undone. You added card Plan."
	what := ""
	if e, err := s.app.Store.Get(chat.ActivityType, r.PathValue("id")); err == nil {
		if said, _ := e.Fields["summary"].(string); said != "" {
			said = chat.CleanSummary(said)
			said = strings.TrimSpace(strings.TrimSuffix(said, "through the API"))
			said = strings.TrimSpace(strings.TrimSuffix(said, "through the command line"))
			what = strings.TrimSuffix(said, ".") + "."
		}
	}
	if err := s.app.Chat.UndoAs("human", r.PathValue("id")); err != nil {
		s.failed(w, r, "Not undone", err, "/")
		return
	}
	s.tell(w, r, outcome{Title: "Undone", Text: what}, "/")
}
