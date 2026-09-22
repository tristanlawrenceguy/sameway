// Package track is the arithmetic of keeping something up: a habit or a
// measure with a cadence (a day, a week), a target for each period (once;
// eight glasses; five kilometres), entries logged against it, and from
// those: what this period holds against its target, the run of periods
// in a row that met it, the best run there has been, and how far a
// longer goal has come. Nothing here is a schema; the habit and entry
// types carry the fields, and this reads them.
package track

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Habit is what is being kept up.
type Habit struct {
	ID, Name, Unit string
	// Cadence is "day" or "week".
	Cadence string
	// Target is what a period needs to count as met; 1 when unset.
	Target float64
	// Goal is a total to reach over time, in the unit; 0 for none.
	Goal float64
}

// Entry is one thing logged: when, and how much.
type Entry struct {
	At     time.Time
	Amount float64
}

// Period is one day or week of a habit, as it went.
type Period struct {
	Start  time.Time
	Amount float64
	Met    bool
}

// Summary is where a habit stands now.
type Summary struct {
	// Now is this period's total, Target what it needs, Met whether it has it.
	Now    float64
	Target float64
	Met    bool
	// Streak is the run of periods in a row that met the target, counting
	// back from this period if it is met and from the one before if not,
	// since a day still going is not a day missed. Best is the longest
	// run there has been.
	Streak, Best int
	// Last is the most recent periods, oldest first, this one last.
	Last []Period
	// Total is everything logged, and Pct how far a goal has come.
	Total float64
	Pct   float64
}

// Summarise reads a habit's entries as of now, with the last n periods.
func Summarise(h Habit, entries []Entry, now time.Time, n int) Summary {
	if h.Target <= 0 {
		h.Target = 1
	}
	if n <= 0 {
		n = 7
	}
	byPeriod := map[time.Time]float64{}
	var total float64
	var first time.Time
	for _, e := range entries {
		start := PeriodStart(e.At, h.Cadence)
		byPeriod[start] += e.Amount
		total += e.Amount
		if first.IsZero() || start.Before(first) {
			first = start
		}
	}
	cur := PeriodStart(now, h.Cadence)
	s := Summary{Target: h.Target, Now: byPeriod[cur], Total: total}
	s.Met = s.Now >= h.Target
	for i := n - 1; i >= 0; i-- {
		start := stepBack(cur, h.Cadence, i)
		amt := byPeriod[start]
		s.Last = append(s.Last, Period{Start: start, Amount: amt, Met: amt >= h.Target})
	}
	// The streak: from this period when met, else from the one before.
	from := cur
	if !s.Met {
		from = stepBack(cur, h.Cadence, 1)
	}
	for p := from; byPeriod[p] >= h.Target; p = stepBack(p, h.Cadence, 1) {
		s.Streak++
		if s.Streak > 100000 {
			break
		}
	}
	// The best run: walk every period from the first entry to now.
	if !first.IsZero() {
		run := 0
		for p := first; !p.After(cur); p = stepForward(p, h.Cadence) {
			if byPeriod[p] >= h.Target {
				run++
				if run > s.Best {
					s.Best = run
				}
			} else {
				run = 0
			}
		}
	}
	if h.Goal > 0 {
		s.Pct = total / h.Goal * 100
		if s.Pct > 100 {
			s.Pct = 100
		}
	}
	return s
}

// PeriodStart is the start of the period a moment falls in: midnight of
// its day, or of the Monday of its week, in local time.
func PeriodStart(t time.Time, cadence string) time.Time {
	t = t.Local()
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	if cadence == "week" {
		return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
	}
	return day
}

func stepBack(start time.Time, cadence string, n int) time.Time {
	if cadence == "week" {
		return start.AddDate(0, 0, -7*n)
	}
	return start.AddDate(0, 0, -n)
}

func stepForward(start time.Time, cadence string) time.Time {
	if cadence == "week" {
		return start.AddDate(0, 0, 7)
	}
	return start.AddDate(0, 0, 1)
}

// Amount is a number as a person reads it: 3, 2.5, 12 km, 8 glasses.
func Amount(v float64, unit string) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if strings.Contains(s, ".") {
		s = strconv.FormatFloat(v, 'f', 1, 64)
		s = strings.TrimSuffix(s, ".0")
	}
	if unit == "" {
		return s
	}
	return s + " " + unit
}

// Progress says where a period stands: "3 of 8 glasses", "done", "1 of 1".
func Progress(s Summary, unit string) string {
	if unit == "" && s.Target == 1 {
		if s.Met {
			return "done"
		}
		return "not yet"
	}
	return Amount(s.Now, "") + " of " + Amount(s.Target, unit)
}

// Streak says the run in words: "3 days in a row", "1 week in a row".
func StreakWords(n int, cadence string) string {
	unit := "day"
	if cadence == "week" {
		unit = "week"
	}
	if n == 1 {
		return fmt.Sprintf("1 %s in a row", unit)
	}
	return fmt.Sprintf("%d %ss in a row", n, unit)
}
