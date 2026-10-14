package agenda

import (
	"net/http"
	"strconv"
	"time"
)

// The days ahead, for an agent. A team lead on a Sunday evening asks to
// plan the week; an agent then rebuilt it from a condition per type, each
// a little differently from Today. GET /api/today?days=7 adds ahead: each
// of the next days as Today will show it on the day (what is due, the
// deadlines, the follow-ups, the events and reminders), so the week is
// read in the same words as the day. What is late is today's alone.

// maxAhead is the most days ahead one answer holds: a month.
const maxAhead = 31

// apiAheadDay is one day ahead.
type apiAheadDay struct {
	Day       string       `json:"day"`
	Date      string       `json:"date"`
	Due       []apiDayItem `json:"due"`
	Deadlines []apiDayItem `json:"deadlines"`
	FollowUps []apiDayItem `json:"follow_ups"`
	Events    []apiDayItem `json:"events"`
	Reminders []apiDayItem `json:"reminders"`
}

// ahead is the days after now's, as many as days asks (none when it does
// not), each as Today would list it on the day.
func (s *Service) ahead(r *http.Request, now time.Time, items func([]todayItem) []apiDayItem) []apiAheadDay {
	n, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if n <= 1 {
		return nil
	}
	n = min(n, maxAhead)
	out := make([]apiAheadDay, 0, n-1)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	for i := 1; i < n; i++ {
		d := start.AddDate(0, 0, i)
		l := s.today(d) // what is late on that day is today's late, said once
		out = append(out, apiAheadDay{Day: d.Format("Monday 2 January"), Date: d.Format("2006-01-02"),
			Due: items(l.Tasks), Deadlines: items(l.Deadlines), FollowUps: items(l.FollowUps), Events: items(l.Events), Reminders: items(l.Reminders)})
	}
	return out
}
