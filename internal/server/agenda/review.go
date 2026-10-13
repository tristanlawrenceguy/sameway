package agenda

import (
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A list kept for weeks fills with what is overdue, and nobody looks back
// to see what went well or decides what to do about the rest: the habit
// that keeps a list worth keeping. The weekly review is one page for it:
// what was done this week first, then what is overdue, each moved to next
// week, kept without a date or ticked in one press, the habits kept, and
// the week ahead with what is on the calendar; and a button that asks the
// assistant to look back and plan with the person. On the day the owner
// picks on Help, a notification says it is ready (review_day.go). Why it
// works this way: design/foundations/review.md.

// reviewTask is a task as the review shows it.
type reviewTask struct {
	ID, Title, When string
	At              time.Time
}

func (s *Service) reviewTasks(now time.Time) (done, overdue, ahead []reviewTask) {
	t, ok := s.app.Types.Get("task")
	if !ok {
		return
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	week := today.AddDate(0, 0, -7)
	recs, _ := s.app.Store.List("task", store.ListOptions{})
	for _, r := range recs {
		item := reviewTask{ID: r.ID, Title: s.Title(t, r)}
		v, _ := r.Fields["due"].(string)
		at, _, has := when.Stored(v, now)
		if v != "" && has {
			item.At = at
			item.When = when.Day(item.At, now)
		}
		isDone, _ := r.Fields["done"].(bool)
		switch {
		case isDone && r.UpdatedAt.After(week):
			done = append(done, item)
		case !isDone && has && item.At.Before(today):
			overdue = append(overdue, item)
		case !isDone && has && item.At.Before(today.AddDate(0, 0, 8)):
			ahead = append(ahead, item)
		}
	}
	for _, l := range [][]reviewTask{done, overdue, ahead} {
		sort.SliceStable(l, func(i, j int) bool { return l[i].At.Before(l[j].At) })
	}
	return
}

// reviewActions are what can be done with an overdue task, each in one
// press: put it off a week, keep it with no date (for some day, as GTD's
// weekly review renegotiates a commitment), or say it is done.
var reviewActions = []struct{ to, label string }{{"next", "Move to next week"}, {"undate", "Clear the date"}, {"done", "Done"}}

func (s *Service) reviewPage(w http.ResponseWriter, r *http.Request) {
	now := s.Now()
	done, overdue, ahead := s.reviewTasks(now)
	esc := template.HTMLEscapeString
	var b strings.Builder
	list := func(items []reviewTask, actions bool) {
		b.WriteString(`<ul class="sw-plain sw-rows">`)
		for _, it := range items {
			// A row as a list's row is: the title, its day at the right,
			// then the presses, which wrap under it on a narrow screen.
			b.WriteString(`<li class="sw-row"><span class="sw-row__title"><a class="sw-row__link" href="/t/task/` + it.ID + `">` + esc(it.Title) + `</a></span>`)
			if it.When != "" {
				b.WriteString(`<span class="sw-row__meta">` + esc(it.When) + `</span>`)
			}
			if actions {
				b.WriteString(`<div class="sw-cluster">`)
				for _, a := range reviewActions {
					b.WriteString(string(s.Form(ui.Form{Action: "/review/" + a.to, Hidden: ui.Hidden("id", it.ID), Button: &ui.Button{Label: a.label, Context: it.Title, Variant: ui.Quiet}})))
				}
				b.WriteString(`</div>`)
			}
			b.WriteString(`</li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`<h2>Done this week</h2>`)
	if len(done) == 0 {
		b.WriteString(`<p class="sw-muted">Nothing ticked off this week yet.</p>`)
	} else {
		list(done, false)
	}
	if len(overdue) > 0 {
		b.WriteString(`<h2>Overdue</h2><p>Their day has passed. Move each to next week, clear its date to keep it for later, or say it is done.</p>`)
		list(overdue, true)
	}
	if habits := blocks.Resolve(s.app.Blocks, blocks.TrackerComponent, map[string]any{"label": "Habits", "level": 2}, blocks.Place{}); hasHabits(habits) {
		b.WriteString(string(s.Component(blocks.TrackerComponent, habits)))
	}
	b.WriteString(`<h2>The week ahead</h2>`)
	if len(ahead) == 0 {
		b.WriteString(`<p class="sw-muted">No tasks are due in the next seven days.</p>`)
	} else {
		list(ahead, false)
	}
	b.WriteString(string(s.reviewCalendar(now)))
	b.WriteString(`<h2>Plan next week</h2><p>The assistant can go through it with you: what went well, what got in the way, and what matters next.</p>`)
	b.WriteString(`<p><a class="sw-link sw-link--button" href="/chat?prompt=` + template.URLQueryEscaper(reviewPrompt) + `">Look back and plan with the assistant</a></p>`)
	s.Page(w, r, "Weekly review", template.HTML(b.String()), web.PageOptions{Lede: template.HTML(esc(reviewSummary(now, len(done), len(overdue), len(ahead))))})
}

// reviewPrompt starts the assistant on the review as a retrospective
// does: what went well first, then what got in the way, then what next,
// without blame.
const reviewPrompt = "Let's do my weekly review. Start with what went well this week, then what got in the way, then help me choose what matters next week from what is overdue and due."

// reviewSummary is the week in one sentence, under the title: the done
// first, the counts in words, and what is none left out, so a screen
// reader hears the whole week once before the lists.
func reviewSummary(now time.Time, done, overdue, ahead int) string {
	said := "The seven days to " + now.Format("Monday 2 January")
	var parts []string
	if done > 0 {
		parts = append(parts, schema.Count(done, "task")+" done")
	}
	if overdue > 0 {
		parts = append(parts, strconv.Itoa(overdue)+" overdue")
	}
	if ahead > 0 {
		parts = append(parts, strconv.Itoa(ahead)+" due in the week ahead")
	}
	switch len(parts) {
	case 0:
		return said + "."
	case 1:
		return said + ": " + parts[0] + "."
	}
	return said + ": " + strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1] + "."
}

// reviewCalendar is what is on the calendar in the next seven days, for
// planning the week around it, as GTD's review looks at the calendar
// ahead; nothing when there is none. A calendar resolves one month, so
// a week that crosses into the next asks for both.
func (s *Service) reviewCalendar(now time.Time) template.HTML {
	if _, ok := s.app.Types.Get("event"); !ok {
		return ""
	}
	from, to := now.Format("2006-01-02"), now.AddDate(0, 0, 7).Format("2006-01-02")
	var events []any
	for _, month := range []string{from[:7], to[:7]} {
		props := blocks.Resolve(s.app.Blocks, blocks.CalendarComponent, map[string]any{"type": "event", "month": month}, blocks.Place{})
		evs, _ := props["events"].([]any)
		for _, e := range evs {
			ev, _ := e.(map[string]any)
			if day, _ := ev["date"].(string); day != "" && day >= from && day <= to {
				events = append(events, ev)
			}
		}
		if from[:7] == to[:7] {
			break
		}
	}
	if len(events) == 0 {
		return ""
	}
	return `<h3>On the calendar</h3>` + s.Component(blocks.CalendarComponent, map[string]any{
		"detail": "brief", "month": from[:7], "today": from, "upcoming": 20, "events": events})
}

// reviewMove moves an overdue task a week on from today; reviewUndate
// keeps it with no date; reviewDone ticks it. Each is the person's
// change, with its Undo.
func (s *Service) reviewMove(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	next := s.Now().AddDate(0, 0, 7).Format("2006-01-02")
	s.reviewWrite(w, r, map[string]any{"due": next}, "Moved to next week")
}

func (s *Service) reviewUndate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	s.reviewWrite(w, r, map[string]any{"due": nil}, "Kept, with no date")
}

func (s *Service) reviewDone(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	s.reviewWrite(w, r, map[string]any{"done": true}, "Done")
}

func (s *Service) reviewWrite(w http.ResponseWriter, r *http.Request, fields map[string]any, said string) {
	id := r.PostForm.Get("id")
	rec, act, err := records.WriteAs(s.app.Store, s.Who(r), "updated", "task", id, fields)
	if err != nil {
		s.Failed(w, r, "Not changed", err, "/review")
		return
	}
	t, _ := s.app.Types.Get("task")
	s.TellAt(w, r, web.Outcome{Title: said, Text: s.Title(t, rec), Undo: act}, "/review")
}

// hasHabits is whether a resolved tracker has a habit to show: a review
// says nothing of habits when none are kept.
func hasHabits(tracker map[string]any) bool {
	h, _ := tracker["habits"].([]any)
	return len(h) > 0
}
