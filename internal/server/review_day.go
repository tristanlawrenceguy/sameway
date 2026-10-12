package server

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The day the weekly review says it is ready: chosen by the owner on
// Help, said once that evening, and never a count of what is late, which
// on a lock screen reads as a scolding (design/foundations/review.md).

// KeepReview says the weekly review is ready on the day review.on names,
// at six in the evening, once that day, while ctx lasts.
func (s *Server) KeepReview(ctx context.Context) {
	go func() {
		tick := time.NewTicker(time.Minute)
		defer tick.Stop()
		for {
			s.reviewIfDue(s.now())
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
}

func (s *Server) reviewIfDue(now time.Time) bool {
	on := strings.ToLower(s.app.Workspace.Config.Review.On)
	if on == "" || s.notify == nil || strings.ToLower(now.Weekday().String()) != on || now.Hour() < 18 || s.app.Store.Meta("review:sent") == now.Format("2006-01-02") {
		return false
	}
	s.app.Store.SetMeta("review:sent", now.Format("2006-01-02"))
	done, _, _ := s.reviewTasks(now)
	go s.notify("Your week in review", reviewNotice(len(done)), s.linkTo("/review"))
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

// reviewLine is the review's line on Help, for the owner: what the
// notification is and when it comes, before it is asked for.
func (s *Server) reviewLine() string {
	on := s.app.Workspace.Config.Review.On
	said := `Weekly review: what was done, what is overdue and the week ahead, at <a class="sw-link" href="/review">Weekly review</a> any time.`
	if on != "" {
		said += " Each " + template.HTMLEscapeString(capitalize(on)) + " at " + s.reviewHour() + ", a notification says it is ready."
	}
	return said + " " + string(s.form(ui.Form{Action: "/review/on", Class: "sw-stack",
		Body: s.component("select", map[string]any{"label": "Remind me to review on", "name": "on", "id": "review-on", "as": "dropdown", "value": on,
			"options": reviewDays()}),
		Button: &ui.Button{Label: "Set the review day", Variant: ui.Secondary}}))
}

func (s *Server) reviewOn(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	on := strings.ToLower(strings.TrimSpace(r.PostForm.Get("on")))
	if s.app.Records.SetSetting == nil {
		s.failed(w, r, "Not set", errors.New("this workspace has no settings file"), "/help")
		return
	}
	if err := s.app.Records.SetSetting("review.on", on); err != nil {
		s.failed(w, r, "Not set", err, "/help")
		return
	}
	o := outcome{Title: "Review day set", Text: "Each " + capitalize(on) + " at " + s.reviewHour() + ", a notification says the weekly review is ready."}
	if on == "" {
		o = outcome{Title: "No review reminder", Text: "Weekly review is there whenever you want it."}
	}
	s.tell(w, r, o, "/help")
}

// reviewHour is when the notice comes, on the person's clock: 6pm, 18:00.
func (s *Server) reviewHour() string {
	return when.Clock(time.Date(2026, 1, 1, 18, 0, 0, 0, time.UTC), s.h24())
}

// reviewDays are the days to choose from, and none.
func reviewDays() []any {
	out := []any{map[string]any{"value": "", "label": "No reminder"}}
	for _, d := range []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday} {
		out = append(out, map[string]any{"value": strings.ToLower(d.String()), "label": d.String()})
	}
	return out
}
