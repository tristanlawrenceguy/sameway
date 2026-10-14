package agenda

import (
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Today for an agent. Asked to help a person decide what today is for,
// the crew's agents rebuilt it from conditions (due<today, due<=+1d,
// done=false), each a little differently, and none with what the person's
// own Today page holds: what is late, what is due, what is on, and the
// three it says to start with, the assistant's own choice among them
// (today_focus.go). GET /api/today is that page as JSON: the same lists in
// the same order, so an agent and the person look at one day.

// apiDayItem is one thing on the day, as an agent reads it.
type apiDayItem struct {
	Type    string `json:"type"`
	ID      string `json:"id"`
	Title   string `json:"title"`
	Page    string `json:"page"`
	When    string `json:"when,omitempty"`
	At      string `json:"at,omitempty"`
	AllDay  bool   `json:"all_day,omitempty"`
	Matters bool   `json:"matters,omitempty"`
}

func (s *Service) todayAPI(w http.ResponseWriter, r *http.Request) {
	now := s.Now()
	l := s.today(now)
	items := func(in []todayItem) []apiDayItem {
		out := make([]apiDayItem, 0, len(in))
		for _, it := range in {
			d := apiDayItem{Type: it.Type, ID: it.ID, Title: it.Title, Page: it.Href, When: it.When, AllDay: it.AllDay, Matters: it.Soon || it.Type == "task" && s.important(it)}
			if !it.At.IsZero() {
				d.At = it.At.Format(time.RFC3339)
			}
			out = append(out, d)
		}
		return out
	}
	answer := map[string]any{
		"day":        now.Format("Monday 2 January"),
		"page":       "/today",
		"late":       items(l.Late),
		"due":        items(l.Tasks),
		"events":     items(l.Events),
		"reminders":  items(l.Reminders),
		"deadlines":  items(l.Deadlines),
		"follow_ups": items(l.FollowUps),
		"count":      l.count(),
		"habits":     s.habitsToDo(now), // today_habits.go: daily ones not met yet, logged by POST /habit/{id}/log
	}
	// A long day starts with three, as the page does; the rest wait.
	if first, rest := s.focus(l); first != nil {
		answer["start_with"] = items(first)
		answer["rest"] = len(rest)
		answer["start_with_why"] = "what the person or the assistant marked as mattering first (tagged important, urgent or start), then what is latest; the page shows these three and folds the rest"
	}
	if days := s.ahead(r, now, items); days != nil { // today_ahead.go
		answer["ahead"] = days
	}
	web.WriteJSON(w, http.StatusOK, answer)
}
