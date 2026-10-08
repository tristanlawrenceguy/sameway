package when

import (
	"testing"
	"time"
)

// A month, a day's heading, a short date and when a message was sent are
// said one way, the year only when it is not this one.
func TestHeadingsAndShortDates(t *testing.T) {
	now := time.Date(2026, 10, 7, 15, 0, 0, 0, time.Local)
	for _, c := range []struct{ got, want string }{
		{Month("2026-09"), "September 2026"},
		{Month("soon"), "soon"},
		{DayHeading(now.Add(-time.Hour), now), "Today, Wednesday 7 October"},
		{DayHeading(now.AddDate(0, 0, -1), now), "Yesterday, Tuesday 6 October"},
		{DayHeading(now.AddDate(0, 0, -3), now), "Sunday 4 October"},
		{DayHeading(now.AddDate(-1, 0, 0), now), "Tuesday 7 October 2025"},
		{Date(now.AddDate(0, -1, 0), now), "7 Sep"},
		{Date(now.AddDate(-1, 0, 0), now), "7 Oct 2025"},
		{Sent(now.Add(-time.Hour), now), Clock(now.Add(-time.Hour))},
		{Sent(now.AddDate(0, 0, -2), now), "5 Oct at " + Clock(now)},
	} {
		if c.got != c.want {
			t.Errorf("got %q, want %q", c.got, c.want)
		}
	}
}
