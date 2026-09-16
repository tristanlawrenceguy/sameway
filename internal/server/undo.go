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
	back := "/"
	if from := r.PostForm.Get("from"); strings.HasPrefix(from, "/") && !strings.HasPrefix(from, "//") {
		back = from
	}
	if err := s.app.Chat.UndoAs("human", r.PathValue("id")); err != nil {
		chat.Record(s.app.Store, "system", chat.Change{Action: "failed", Detail: "undo: " + err.Error()})
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// undoable is a receipt with Undo only where undo is a moment: under the
// newest reply, and only for changes that can still be reversed. Older
// receipts keep their links and lose the control; the activity page has it.
func (s *Server) undoable(changes any, latest bool) any {
	list, ok := changes.([]any)
	if !ok {
		return changes
	}
	out := make([]any, 0, len(list))
	for _, item := range list {
		c, ok := item.(map[string]any)
		if !ok {
			continue
		}
		copied := map[string]any{}
		for k, v := range c {
			copied[k] = v
		}
		if id, _ := c["activity"].(string); id != "" {
			keep := false
			if latest {
				if entry, err := s.app.Store.Get(chat.ActivityType, id); err == nil {
					keep = s.app.Chat.Undoable(entry)
				}
			}
			if !keep {
				delete(copied, "activity")
			}
		}
		out = append(out, copied)
	}
	return out
}
