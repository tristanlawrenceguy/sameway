package inbox

import (
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A bill comes as a task to pay, suggested from the email or the photo of
// it. Its reminder, and the next email about the same bill, are nothing
// new: the task suggested from them changes the one still open, with a
// new amount or day, rather than making a second thing to pay.

// tasksChangeOneKept has the tasks suggested from email and from what is
// shared change one still open, once, for sorting set up before they did.
func (s *Service) tasksChangeOneKept() {
	const set = "sorting:tasks-change-one-kept"
	if s.app.Store.Meta(set) != "" || s.app.Store.Meta(sortingSetUp) == "" {
		return
	}
	s.app.Store.SetMeta(set, "yes")
	who := records.Who{Actor: "system", Via: "setting up sorting"}
	actions, _ := s.app.Store.List(records.ActionType, store.ListOptions{})
	for _, a := range actions {
		if a.Fields["kind"] == "suggest" && a.Fields["make"] == "task" && (a.Fields["what"] == "email" || a.Fields["what"] == "note") {
			records.WriteAs(s.app.Store, who, "updated", records.ActionType, a.ID, map[string]any{"update": true})
		}
	}
}
