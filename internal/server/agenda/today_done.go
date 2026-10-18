package agenda

import (
	"html/template"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// At the end of the day a person wants to see what got done, and went
// looking for it through the task list, the activity log and each page in
// turn. Today says it in a line, under what is still to do: the tasks
// ticked today, the daily habits met today (which leave Today's habits
// once met, so a carer never saw all three doses given), and the calls
// and visits that happened today, by name. Nothing is said before
// anything is done.

// doneToday is what got done today: tasks ticked, the latest first, then
// daily habits met, then dealings with people that happened today.
func (s *Service) doneToday(now time.Time) []string {
	out := []string{}
	day := now.Format("2006-01-02")
	if _, ok := s.app.Types.Get("task"); ok {
		recs, _ := s.app.Store.List("task", store.ListOptions{OrderBy: "updated_at", Desc: true})
		for _, r := range recs {
			if done, _ := r.Fields["done"].(bool); done && r.UpdatedAt.In(now.Location()).Format("2006-01-02") == day {
				title, _ := r.Fields["title"].(string)
				out = append(out, title)
			}
		}
	}
	out = append(out, s.habitsMet(now)...)
	return append(out, s.dealtWithToday(now)...)
}

// habitsMet is the daily habits to reach that were met today, by name.
func (s *Service) habitsMet(now time.Time) []string {
	if _, ok := s.app.Types.Get(records.HabitType); !ok {
		return nil
	}
	recs, _ := s.app.Store.List(records.HabitType, store.ListOptions{OrderBy: "name"})
	var out []string
	for _, r := range recs {
		h := track.Normal(blocks.HabitOf(r))
		if archived, _ := r.Fields["archived"].(bool); archived || h.Cadence != "day" || h.Aim != track.Reach {
			continue
		}
		if track.Summarise(h, blocks.EntriesOf(s.app.Store, h.ID), now, 1).Met {
			out = append(out, h.Name)
		}
	}
	return out
}

// dealtWithToday is the interactions that happened today, by summary.
func (s *Service) dealtWithToday(now time.Time) []string {
	if _, ok := s.app.Types.Get("interaction"); !ok {
		return nil
	}
	recs, _ := s.app.Store.List("interaction", store.ListOptions{})
	day := now.Format("2006-01-02")
	var out []string
	for _, r := range recs {
		v, _ := r.Fields["at"].(string)
		if at, _, ok := when.Stored(v, now); ok && !at.After(now) && at.Format("2006-01-02") == day {
			summary, _ := r.Fields["summary"].(string)
			out = append(out, summary)
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
