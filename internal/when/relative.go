package when

import (
	"fmt"
	"strings"
	"time"
)

// Relative is a stored moment or day the way a person says it about now:
// "Today at 2:30pm", "Yesterday at 9am", "4 days ago at 1:31pm". Day-only
// values fall through to Short-like behaviour (Today, Tomorrow, weekday).
// Anything that is not a stored value comes back as it is.
func Relative(v string, now time.Time) string {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return v
	}
	dayOnly := strings.HasSuffix(v, "T00:00:00Z")

	// Day-only values: fall through to Short-like behaviour.
	if dayOnly {
		return shortDay(ts, now)
	}

	// Moment with time-of-day: compute relative phrasing in UTC (the zone
	// the store uses), then format clock for display in local time.
	nowUTC := now.UTC()
	today := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	local := ts.UTC()
	d := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	diff := int(d.Sub(today).Hours() / 24)

	// Use UTC for the clock display: stored values are always in UTC and this
	// avoids timezone-shift bugs when Local() moves times by hours (e.g. on Windows
	// boxes where the system zone is not UTC).
	clock := formatClock(local)

	switch {
	case diff == 0:
		return "Today at " + clock
	case diff < 0 && diff >= -6: // past within a week (today handled above)
		if diff == -1 {
			return "Yesterday at " + clock
		}
		return fmt.Sprintf("%d days ago at %s", -diff, clock)
	case diff > 0 && diff <= 6: // future within a week
		if diff == 1 {
			return "Tomorrow at " + clock
		}
		return fmt.Sprintf("In %d days at %s", diff, clock)
	default:
		// Older dates.
		if local.Year() == nowUTC.Year() {
			return d.Format("Mon 2 Jan") + " at " + clock
		}
		return fmt.Sprintf("%s at %s %d", d.Format("Jan 2"), clock, local.Year())
	}
}

// shortDay formats a day-only value like Short does: Today, Tomorrow,
// weekday within the week, else "Day Month" or "Day Month Year".
func shortDay(ts time.Time, now time.Time) string {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	d := time.Date(ts.UTC().Year(), ts.UTC().Month(), ts.UTC().Day(), 0, 0, 0, 0, now.Location())
	diff := int(d.Sub(today).Hours() / 24)

	switch {
	case diff == 0:
		return "Today"
	case diff < 0 && diff >= -6: // past within a week (today handled above)
		if diff == -1 {
			return "Yesterday"
		}
		return d.Format("Mon 2 Jan")
	case diff > 0 && diff <= 6: // future within a week
		if diff == 1 {
			return "Tomorrow"
		}
		return fmt.Sprintf("In %d days", diff)
	default:
		if d.Year() == now.Year() {
			return d.Format("2 Jan")
		}
		return d.Format("2 Jan 2006")
	}
}

// formatClock formats a local time as 12-hour "H:MMam/pm".
func formatClock(t time.Time) string {
	h := t.Hour()
	min := t.Minute()
	suffix := "am"
	if h >= 12 {
		suffix = "pm"
	}
	h12 := h % 12
	if h12 == 0 {
		h12 = 12
	}
	return fmt.Sprintf("%d:%02d%s", h12, min, suffix)
}
