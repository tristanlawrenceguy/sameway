package server

import (
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
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
			what = strings.TrimSuffix(said, ".") + "."
		}
	}
	if err := s.app.Chat.UndoAs("human", r.PathValue("id")); err != nil {
		s.failed(w, r, "Not undone", err, "/")
		return
	}
	s.tell(w, r, outcome{Title: "Undone", Text: what}, "/")
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

// humanizeChanges converts machine-language field names in a changes list into
// human-readable descriptions for setting changes only — other changes pass
// through unchanged. It mutates each item's "action", "component" and "detail".
func (s *Server) humanizeChanges(changes any) any {
	list, ok := changes.([]any)
	if !ok {
		return changes
	}
	out := make([]any, 0, len(list))
	for _, item := range list {
		c, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		copied := map[string]any{}
		for k, v := range c {
			copied[k] = v
		}
		if action, _ := copied["action"].(string); action == "set" {
			component, _ := copied["component"].(string)
			detail, _ := copied["detail"].(string)
			if label, ok := workspace.SettingTarget(component); ok {
				copied["action"] = "changed"
				_ = label // used by SettingTarget to confirm it's a known setting key
				copied["component"] = ""
				copied["detail"] = workspace.SettingValueLabel(component, detail)
			}
		}
		out = append(out, copied)
	}
	return out
}
