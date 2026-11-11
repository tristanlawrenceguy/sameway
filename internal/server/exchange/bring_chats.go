package exchange

import (
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A year of conversations brought from ChatGPT or Claude is hundreds of
// notes, and what was still to do in them (a van to book on the 14th, a
// call to return) stays words nobody reads again. When the person asks, an
// action is set up before they come in: each conversation brought, as it
// is added, is read for what is still to do, and that is put on Today as a
// task to make, change or turn down (automate/suggest.go), learning from
// what they did with the ones before. It is an action like any other, on
// Actions, to change or remove; nothing is made without a press.

// suggestFromChats sets up the action that suggests a task from each
// conversation tagged tag as it is added, unless one is there already.
func (s *Service) suggestFromChats(r *http.Request, tag string) {
	actions, _ := s.app.Store.List(records.ActionType, store.ListOptions{})
	for _, a := range actions {
		if a.Fields["kind"] == "suggest" && a.Fields["what"] == "note" {
			for _, o := range records.StringList(a.Fields["only"]) {
				if o == "tags="+tag {
					return
				}
			}
		}
	}
	records.WriteAs(s.app.Store, s.Who(r), "created", records.ActionType, "", map[string]any{
		"title": "Suggest a task from each conversation brought from " + map[string]string{"chatgpt": "ChatGPT", "claude": "Claude"}[tag],
		"kind":  "suggest", "make": "task", "when": "added", "what": "note", "only": []any{"tags=" + tag},
		"check": true, "examples": 10,
	})
}
