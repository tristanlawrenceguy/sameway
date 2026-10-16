package agenda

import (
	"html/template"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// At the end of the day a person wants to see what got done, and went
// looking for it through the task list, the activity log and each page in
// turn. Today says it in a line, under what is still to do: the tasks
// ticked today, by name. Nothing is said before anything is done.

// doneToday is the tasks ticked today, the latest first.
func (s *Service) doneToday(now time.Time) []string {
	if _, ok := s.app.Types.Get("task"); !ok {
		return nil
	}
	recs, _ := s.app.Store.List("task", store.ListOptions{OrderBy: "updated_at", Desc: true})
	day := now.Format("2006-01-02")
	out := []string{}
	for _, r := range recs {
		if done, _ := r.Fields["done"].(bool); done && r.UpdatedAt.In(now.Location()).Format("2006-01-02") == day {
			title, _ := r.Fields["title"].(string)
			out = append(out, title)
		}
	}
	return out
}

// doneLine is Today's line of what was done today, or "".
func doneLine(done []string) string {
	if len(done) == 0 {
		return ""
	}
	return `<p class="sw-muted">Done today: ` + template.HTMLEscapeString(schema.AndNames(done)) + `.</p>`
}
