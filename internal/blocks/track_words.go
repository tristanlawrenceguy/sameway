package blocks

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/track"
)

// A habit's periods in words: which one it is, as a person says a date, and
// how it went. A period still going is not one missed.

// periodName is a period as a person names it: Wed 16 Sep, week of 14 Sep,
// September, 2026.
func periodName(start time.Time, cadence string) string {
	switch cadence {
	case "week":
		return "week of " + start.Format("2 Jan")
	case "month":
		return start.Format("January 2006")
	case "year":
		return start.Format("2006")
	}
	return start.Format("Mon 2 Jan")
}

// periodWhen is when a habit's standing is for, after its name: today, this
// week, or, for a day logged for in the past, on Mon 21 Sep.
func periodWhen(now, today time.Time, cadence string) string {
	if track.PeriodStart(now, cadence).Equal(track.PeriodStart(today, cadence)) {
		if cadence == "day" || cadence == "" {
			return "today"
		}
		return "this " + cadence
	}
	if cadence == "day" || cadence == "" {
		return "on " + now.Format("Mon 2 Jan")
	}
	return "in the " + periodName(track.PeriodStart(now, cadence), cadence)
}

// periodWords is how a period went, for a reader of the dots: met or not
// met, within or over a limit, or what was recorded; the one still going
// says how far it has got, not that it was missed.
func periodWords(h track.Habit, p track.Period, going bool, sum track.Summary) string {
	switch h.Aim {
	case track.Record:
		if !p.Logged {
			return "nothing logged"
		}
		return track.Amount(p.Amount, h.Unit)
	case track.Limit:
		if p.Met {
			return track.Amount(p.Amount, h.Unit) + ", within the limit"
		}
		if p.Amount > h.Target {
			return track.Amount(p.Amount, h.Unit) + ", over the limit"
		}
		return "not tracked yet"
	}
	switch {
	case going && !p.Met:
		return "still going, " + track.Progress(h, sum) + " so far"
	case p.Met:
		return ""
	case !p.Logged:
		return "nothing logged"
	}
	return track.Amount(p.Amount, "") + " of " + track.Amount(h.Target, h.Unit)
}

// missedOnce is what a habit aimed to reach says the period after one
// miss, before this one is met: "Missed yesterday; today gets back on
// track". One miss does not undo a habit; two in a row starts to, so the
// row says so then and looks ahead, never "0 days". After two, it says
// nothing: the page has the best run and how often it was reached.
func missedOnce(h track.Habit, periods []track.Period, sum track.Summary) string {
	n := len(periods)
	if h.Aim != track.Reach || sum.Met || n < 3 || periods[n-2].Met || !periods[n-3].Met {
		return ""
	}
	if h.Cadence == "day" {
		return "Missed yesterday; today gets back on track"
	}
	return "Missed last " + h.Cadence + "; this " + h.Cadence + " gets back on track"
}

// trackerShows is a tracker in a few words: how many habits, and which.
func trackerShows(_ *Workspace, props, out map[string]any) string {
	habits, _ := out["habits"].([]any)
	shows := schema.Count(len(habits), records.HabitType)
	if tags := Strs(props["tags"]); len(tags) > 0 {
		shows += " tagged " + strings.Join(tags, " or ")
	}
	if len(Strs(props["habits"])) > 0 {
		var names []string
		for _, h := range habits {
			if m, ok := h.(map[string]any); ok {
				names = append(names, str(m["name"], ""))
			}
		}
		shows += ": " + AndList(names)
	}
	return shows
}
