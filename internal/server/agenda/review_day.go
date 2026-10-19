package agenda

import (
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The day the weekly review says it is ready: chosen by the owner on
// Help, said once that evening, and never a count of what is late, which
// on a lock screen reads as a scolding (design/foundations/review.md).

// ReviewNext is when the weekly review is next ready after now: six in
// the evening on the day review.on names, read each time so a change
// applies. It is the zero time when no day is set.
func (s *Service) ReviewNext(now time.Time) time.Time {
	on := strings.ToLower(s.app.Workspace.Config.Review.On)
	for d := range 8 {
		t := time.Date(now.Year(), now.Month(), now.Day()+d, 18, 0, 0, 0, now.Location())
		if t.After(now) && strings.ToLower(t.Weekday().String()) == on {
			return t
		}
	}
	return time.Time{}
}

// ReviewIfDue says the review is ready when it is the day, the evening
// has come and it has not been said today, and says whether it did.
func (s *Service) ReviewIfDue(now time.Time) bool {
	on := strings.ToLower(s.app.Workspace.Config.Review.On)
	if on == "" || !s.CanNotify() || strings.ToLower(now.Weekday().String()) != on || now.Hour() < 18 || s.app.Store.Meta("review:sent") == now.Format("2006-01-02") {
		return false
	}
	s.app.Store.SetMeta("review:sent", now.Format("2006-01-02"))
	done, _, _ := s.reviewTasks(now)
	s.Notify("Your week in review", reviewNotice(len(done)), "/review")
	return true
}

// reviewNotice is the notification's words: what was done when there is
// some, then the invitation. What is overdue waits on the page, with the
// ways to deal with it beside it.
func reviewNotice(done int) string {
	if done == 0 {
		return "Look back on the week and plan the next one."
	}
	return schema.Count(done, "task") + " done this week. Look back, and plan the next one."
}

// ReviewLine is the review's line on Help, for the owner: what the
// notification is and when it comes, before it is asked for.
func (s *Service) ReviewLine() string {
	on := s.app.Workspace.Config.Review.On
	said := `Weekly review: what was done, what is overdue and the week ahead, at <a class="sw-link" href="/review">Weekly review</a> any time.`
	if on != "" {
		said += " Each " + template.HTMLEscapeString(blocks.Capitalize(on)) + " at " + s.reviewHour() + ", a notification says it is ready."
	}
	return said + " " + string(s.Form(ui.Form{Action: "/review/on", Class: "sw-stack",
		Body: s.Component("select", map[string]any{"label": "Remind me to review on", "name": "on", "id": "review-on", "as": "dropdown", "value": on,
			"options": reviewDays()}),
		Button: &ui.Button{Label: "Set the review day", Variant: ui.Secondary}}))
}

func (s *Service) reviewOn(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	on := strings.ToLower(strings.TrimSpace(r.PostForm.Get("on")))
	if s.app.Records.SetSetting == nil {
		s.Failed(w, r, "Not set", errors.New("this workspace has no settings file"), "/help")
		return
	}
	if err := s.app.Records.SetSetting("review.on", on); err != nil {
		s.Failed(w, r, "Not set", err, "/help")
		return
	}
	o := web.Outcome{Title: "Review day set", Text: "Each " + blocks.Capitalize(on) + " at " + s.reviewHour() + ", a notification says the weekly review is ready."}
	if on == "" {
		o = web.Outcome{Title: "No review reminder", Text: "Weekly review is there whenever you want it."}
	}
	s.Tell(w, r, o, "/help")
}

// reviewHour is when the notice comes, on the person's clock: 6pm, 18:00.
func (s *Service) reviewHour() string {
	return when.Clock(time.Date(2026, 1, 1, 18, 0, 0, 0, time.UTC), s.H24())
}

// reviewDays are the days to choose from, and none.
func reviewDays() []any {
	out := []any{map[string]any{"value": "", "label": "No reminder"}}
	for _, d := range []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday} {
		out = append(out, map[string]any{"value": strings.ToLower(d.String()), "label": d.String()})
	}
	return out
}

// planLine is Today's line on the day to plan the week (the review day
// the owner chose, or Sunday when none is chosen), and on a long Today,
// where a person back from a move, facing 69 late and due, asked for what
// needs attention this week apart from old overdue noise: the review is
// that page. People planning the week looked for it through the task list,
// the calendar and every page, and never found it, linked from Help.
func (s *Service) planLine(now time.Time, l todayLists) string {
	on := strings.ToLower(s.app.Workspace.Config.Review.On)
	if on == "" {
		on = "sunday"
	}
	if first, _ := s.focus(l); strings.ToLower(now.Weekday().String()) != on && first == nil {
		return ""
	}
	return `<p class="sw-muted">Plan the week: <a class="sw-link" href="/review">Weekly review</a>, what was done, what is overdue and the week ahead.</p>`
}
