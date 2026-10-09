package server

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A list kept for weeks fills with what slipped, and nobody looks back to
// see what went well or decides what to do about the rest: the habit that
// keeps a list worth keeping. The weekly review is one page for it: what
// was done this week, what slipped, with Move to next week or Done for each
// in one press, the habits kept, and the week ahead; and a button that asks
// the assistant to plan the next week with the person. On the day the
// owner picks on Help, a notification says it is ready.

// reviewTask is a task as the review shows it.
type reviewTask struct {
	ID, Title, When string
	At              time.Time
}

func (s *Server) reviewTasks(now time.Time) (done, slipped, ahead []reviewTask) {
	t, ok := s.app.Types.Get("task")
	if !ok {
		return
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	week := today.AddDate(0, 0, -7)
	recs, _ := s.app.Store.List("task", store.ListOptions{})
	for _, r := range recs {
		item := reviewTask{ID: r.ID, Title: s.title(t, r)}
		v, _ := r.Fields["due"].(string)
		at, _, has := when.Parse(v, now)
		if has {
			item.At = at.In(time.Local)
			item.When = when.Day(item.At, now)
		}
		isDone, _ := r.Fields["done"].(bool)
		switch {
		case isDone && r.UpdatedAt.After(week):
			done = append(done, item)
		case !isDone && has && item.At.Before(today):
			slipped = append(slipped, item)
		case !isDone && has && item.At.Before(today.AddDate(0, 0, 8)):
			ahead = append(ahead, item)
		}
	}
	for _, l := range [][]reviewTask{done, slipped, ahead} {
		sort.SliceStable(l, func(i, j int) bool { return l[i].At.Before(l[j].At) })
	}
	return
}

func (s *Server) reviewPage(w http.ResponseWriter, r *http.Request) {
	now := s.now()
	done, slipped, ahead := s.reviewTasks(now)
	esc := template.HTMLEscapeString
	var b strings.Builder
	list := func(items []reviewTask, actions bool) {
		b.WriteString(`<ul class="sw-plain sw-rows">`)
		for _, it := range items {
			b.WriteString(`<li class="sw-cluster"><a class="sw-link" href="/t/task/` + it.ID + `">` + esc(it.Title) + `</a>`)
			if it.When != "" {
				b.WriteString(` <span class="sw-muted sw-small">` + esc(it.When) + `</span>`)
			}
			if actions {
				for _, a := range []struct{ to, label string }{{"next", "Move to next week"}, {"done", "Done"}} {
					b.WriteString(string(s.form(ui.Form{Action: "/review/" + a.to, Hidden: ui.Hidden("id", it.ID), Button: &ui.Button{Label: a.label, Context: it.Title, Variant: ui.Quiet}})))
				}
			}
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`<h2>Done this week</h2>`)
	if len(done) == 0 {
		b.WriteString(`<p class="sw-muted">Nothing ticked off this week yet.</p>`)
	} else {
		b.WriteString(`<p>` + esc(countWords(len(done), "task", "tasks")) + ` done.</p>`)
		list(done, false)
	}
	b.WriteString(`<h2>Slipped</h2>`)
	if len(slipped) == 0 {
		b.WriteString(`<p class="sw-muted">Nothing is late.</p>`)
	} else {
		b.WriteString(`<p>Late, and still to do. Move each to next week, or say it is done.</p>`)
		list(slipped, true)
	}
	b.WriteString(`<h2>Habits</h2>` + string(s.component("tracker", s.tracker(map[string]any{"level": 3}))))
	b.WriteString(`<h2>The week ahead</h2>`)
	if len(ahead) == 0 {
		b.WriteString(`<p class="sw-muted">Nothing is due in the next seven days.</p>`)
	} else {
		list(ahead, false)
	}
	b.WriteString(`<p><a class="sw-link sw-link--button" href="/chat?prompt=` + template.URLQueryEscaper("Let's plan next week: look at what slipped and what is due, and help me decide what matters.") + `">Plan next week with the assistant</a></p>`)
	s.page(w, r, "Weekly review", template.HTML(b.String()), pageOptions{Lede: template.HTML("The week to " + now.Format("Monday 2 January"))})
}

func countWords(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

// reviewMove moves a slipped task a week on from today; reviewDone ticks
// it. Both are the person's change, with its Undo.
func (s *Server) reviewMove(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	next := s.now().AddDate(0, 0, 7).Format("2006-01-02")
	s.reviewWrite(w, r, map[string]any{"due": next}, "Moved to next week")
}

func (s *Server) reviewDone(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	s.reviewWrite(w, r, map[string]any{"done": true}, "Done")
}

func (s *Server) reviewWrite(w http.ResponseWriter, r *http.Request, fields map[string]any, said string) {
	id := r.PostForm.Get("id")
	rec, act, err := records.WriteAs(s.app.Store, s.who(r), "updated", "task", id, fields)
	if err != nil {
		s.failed(w, r, "Not changed", err, "/review")
		return
	}
	t, _ := s.app.Types.Get("task")
	s.tellAt(w, r, outcome{Title: said, Text: s.title(t, rec), Undo: act}, "/review")
}

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
	done, slipped, _ := s.reviewTasks(now)
	go s.notify("Your week in review", countWords(len(done), "task", "tasks")+" done, "+countWords(len(slipped), "slipped", "slipped")+". Plan the week ahead.", s.linkTo("/review"))
	return true
}

// reviewLine is the review's line on Help, for the owner.
func (s *Server) reviewLine() string {
	on := s.app.Workspace.Config.Review.On
	said := `Weekly review: what was done, what slipped and the week ahead, at <a class="sw-link" href="/review">Weekly review</a> any time.`
	if on != "" {
		said += " It says it is ready each " + template.HTMLEscapeString(capitalize(on)) + " evening."
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
	o := outcome{Title: "Review day set", Text: "Each " + capitalize(on) + " evening, the weekly review says it is ready."}
	if on == "" {
		o = outcome{Title: "No review reminder", Text: "Weekly review is there whenever you want it."}
	}
	s.tell(w, r, o, "/help")
}

// reviewDays are the days to choose from, and none.
func reviewDays() []any {
	out := []any{map[string]any{"value": "", "label": "No reminder"}}
	for _, d := range []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday} {
		out = append(out, map[string]any{"value": strings.ToLower(d.String()), "label": d.String()})
	}
	return out
}
