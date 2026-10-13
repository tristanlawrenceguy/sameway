package blocks

import (
	"fmt"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/track"
)

// A habit said in a sentence or two: where it stands, and how it has gone.
// The same words reach the person on its page and an agent through
// get_record, so neither reads the numbers off a picture.

// Since is a habit's last periods from its first entry on. The periods
// before it are before the habit was kept: not a miss, and not counted.
func Since(sum track.Summary) []track.Period {
	if sum.First.IsZero() {
		return nil
	}
	for i, p := range sum.Last {
		if !p.Start.Before(sum.First) {
			return sum.Last[i:]
		}
	}
	return nil
}

// HowKept is how a habit has gone over its last periods, counted from its
// first entry: "Reached the target of 8 glasses on 8 of the 11 days since
// Sat 3 Oct. Best run: 5 days in a row." A rate, not only the run, so a
// day missed does not wipe out what was done. Empty for a record, which
// has no target, and until there is more than the period still going.
func HowKept(h track.Habit, sum track.Summary) string {
	h = track.Normal(h)
	periods := Since(sum)
	if h.Aim == track.Record || len(periods) < 2 {
		return ""
	}
	met := 0
	for _, p := range periods {
		if p.Met {
			met++
		}
	}
	what := "Reached the target of " + track.Amount(h.Target, h.Unit)
	switch {
	case h.Aim == track.Limit:
		what = "Within the limit of " + track.Amount(h.Target, h.Unit)
	case h.Unit == "" && h.Target == 1:
		what = "Done"
	}
	on := "on"
	if h.Cadence != "day" {
		on = "in"
	}
	over := fmt.Sprintf("the last %d %ss", len(periods), h.Cadence)
	if len(periods) < len(sum.Last) {
		from := periodName(sum.First, h.Cadence)
		if h.Cadence == "week" {
			from = "the " + from
		}
		over = fmt.Sprintf("the %d %ss since %s", len(periods), h.Cadence, from)
	}
	s := fmt.Sprintf("%s %s %d of %s.", what, on, met, over)
	if sum.Best > 0 {
		s += " Best run: " + track.StreakWords(sum.Best, h) + "."
	}
	return s
}

// HabitSaid is where a habit stands and how it has gone, as an agent is
// told it: "3 of 8 glasses today. 3 days in a row. Reached the target of
// 8 glasses on 8 of the last 30 days. Best run: 5 days in a row."
func HabitSaid(st *store.Store, rec *store.Record, now time.Time) string {
	h := track.Normal(HabitOf(rec))
	entries := EntriesOf(st, h.ID)
	sum := track.Summarise(h, entries, now, map[string]int{"day": 30, "week": 12, "month": 12, "year": 5}[h.Cadence])
	s := track.Progress(h, sum) + " " + periodWhen(now, now, h.Cadence) + "."
	if sum.Streak > 0 {
		s += " " + track.StreakWords(sum.Streak, h) + "."
	}
	if kept := HowKept(h, sum); kept != "" {
		s += " " + kept
	}
	return s
}
