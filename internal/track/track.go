// Package track is the arithmetic of keeping something up: a habit or a
// measure with a cadence (a day, a week, a month, a year), a target for
// each period, entries logged against it, and from those: what this
// period holds against its target, the run of periods in a row that met
// it, the best run there has been, and how far a longer goal has come.
//
// A target is aimed at one of three ways. To reach it is at least the
// target: eight glasses, five kilometres. To keep within it is at most:
// a budget, an allowance of hours, screen time; what goes past it is the
// over. To record is no target at all, only the numbers. A period's
// entries are combined one of three ways: added up, the latest one (a
// weight), or their average (hours slept). Nothing here is a schema; the
// habit and entry types carry the fields, and this reads them.
package track

import (
	"fmt"
	"time"
)

// The aims.
const (
	Reach  = "reach"
	Limit  = "limit"
	Record = "record"
)

// The ways of combining a period's entries.
const (
	Sum     = "sum"
	Latest  = "latest"
	Average = "average"
)

// Habit is what is being kept up.
type Habit struct {
	ID, Name, Unit string
	// Cadence is "day", "week", "month" or "year"; a day when unset.
	Cadence string
	// Target is what a period is aimed at; 1 when unset and the aim is
	// to reach it, none when the aim is to record.
	Target float64
	// Aim is Reach (at least the target), Limit (at most) or Record (no
	// target); Reach when unset.
	Aim string
	// Combine is Sum, Latest or Average; Sum when unset.
	Combine string
	// Goal is a total to reach over time, in the unit; 0 for none.
	Goal float64
}

// Entry is one thing logged: when, and how much.
type Entry struct {
	At     time.Time
	Amount float64
}

// Period is one day, week, month or year of a habit, as it went. Logged
// says whether anything was, since an empty period of a latest or an
// average has no amount rather than none.
type Period struct {
	Start  time.Time
	Amount float64
	Met    bool
	Logged bool
}

// Summary is where a habit stands now.
type Summary struct {
	// Now is this period's amount, Target what it is aimed at, Met whether
	// it is reached (or, for a limit, kept within), Logged whether anything
	// has been logged this period.
	Now    float64
	Target float64
	Met    bool
	Logged bool
	// Left is what remains before the target (to reach it, or to use of
	// a limit), and Over what a limit has gone past it.
	Left, Over float64
	// Streak is the run of periods in a row that met the target, counting
	// back from this period if it is met and from the one before if not,
	// since a day still going is not a day missed. Best is the longest
	// run there has been. Neither counts before the first entry, and a
	// record has neither.
	Streak, Best int
	// Last is the most recent periods, oldest first, this one last.
	Last []Period
	// Total is everything logged, and Pct how far a goal has come.
	Total float64
	Pct   float64
}

// Normal fills in what a habit leaves unset: a day, to reach, added up,
// and a target of one unless it only records.
func Normal(h Habit) Habit {
	switch h.Cadence {
	case "week", "month", "year":
	default:
		h.Cadence = "day"
	}
	switch h.Aim {
	case Limit, Record:
	default:
		h.Aim = Reach
	}
	switch h.Combine {
	case Latest, Average:
	default:
		h.Combine = Sum
	}
	if h.Aim == Record {
		h.Target = 0
	} else if h.Target <= 0 && h.Aim == Reach {
		h.Target = 1
	}
	return h
}

// meets says whether a period's amount meets the target.
func meets(h Habit, amount float64, logged bool) bool {
	switch h.Aim {
	case Record:
		return false
	case Limit:
		return amount <= h.Target
	}
	return logged && amount >= h.Target
}

// tally is one period's entries as they come in.
type tally struct {
	sum    float64
	count  int
	latest time.Time
	last   float64
}

// Summarise reads a habit's entries as of now, with the last n periods.
func Summarise(h Habit, entries []Entry, now time.Time, n int) Summary {
	h = Normal(h)
	if n <= 0 {
		n = 7
	}
	byPeriod := map[time.Time]*tally{}
	var total float64
	var first time.Time
	for _, e := range entries {
		start := PeriodStart(e.At, h.Cadence)
		t := byPeriod[start]
		if t == nil {
			t = &tally{}
			byPeriod[start] = t
		}
		t.sum += e.Amount
		t.count++
		if t.count == 1 || !e.At.Before(t.latest) {
			t.latest, t.last = e.At, e.Amount
		}
		total += e.Amount
		if first.IsZero() || start.Before(first) {
			first = start
		}
	}
	amountOf := func(p time.Time) (float64, bool) {
		t := byPeriod[p]
		if t == nil {
			return 0, false
		}
		switch h.Combine {
		case Latest:
			return t.last, true
		case Average:
			return t.sum / float64(t.count), true
		}
		return t.sum, true
	}
	// A period counts only from the first entry on: a limit was not kept
	// in the months before anything was tracked.
	metAt := func(p time.Time) bool {
		if first.IsZero() || p.Before(first) {
			return false
		}
		amt, logged := amountOf(p)
		return meets(h, amt, logged)
	}
	cur := PeriodStart(now, h.Cadence)
	s := Summary{Target: h.Target, Total: total}
	s.Now, s.Logged = amountOf(cur)
	s.Met = meets(h, s.Now, s.Logged)
	if h.Aim != Record {
		s.Left = max(0, h.Target-s.Now)
	}
	if h.Aim == Limit {
		s.Over = max(0, s.Now-h.Target)
	}
	for i := n - 1; i >= 0; i-- {
		start := stepBack(cur, h.Cadence, i)
		amt, logged := amountOf(start)
		s.Last = append(s.Last, Period{Start: start, Amount: amt, Met: metAt(start), Logged: logged})
	}
	if h.Aim != Record && !first.IsZero() {
		// The streak: from this period when met, else from the one before.
		from := cur
		if !metAt(cur) {
			from = stepBack(cur, h.Cadence, 1)
		}
		for p := from; metAt(p); p = stepBack(p, h.Cadence, 1) {
			s.Streak++
		}
		// The best run: walk every period from the first entry to now.
		run := 0
		for p := first; !p.After(cur); p = stepBack(p, h.Cadence, -1) {
			if metAt(p) {
				run++
				s.Best = max(s.Best, run)
			} else {
				run = 0
			}
		}
	}
	if h.Goal > 0 {
		s.Pct = min(100, total/h.Goal*100)
	}
	return s
}

// PeriodStart is the start of the period a moment falls in: midnight of
// its day, of the Monday of its week, of the first of its month or of its
// year, in local time.
func PeriodStart(t time.Time, cadence string) time.Time {
	t = t.Local()
	switch cadence {
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case "year":
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	}
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	if cadence == "week" {
		return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
	}
	return day
}

// stepBack is the period n before start; a negative n steps forward.
func stepBack(start time.Time, cadence string, n int) time.Time {
	switch cadence {
	case "week":
		return start.AddDate(0, 0, -7*n)
	case "month":
		return start.AddDate(0, -n, 0)
	case "year":
		return start.AddDate(-n, 0, 0)
	}
	return start.AddDate(0, 0, -n)
}

// PeriodLabel is a period as a chart labels it: the day (a week by its
// Monday), the month, the year.
func PeriodLabel(start time.Time, cadence string) string {
	switch cadence {
	case "month":
		return start.Format("2006-01")
	case "year":
		return start.Format("2006")
	}
	return start.Format("2006-01-02")
}

// StreakWords says the run in words: "3 days in a row", "1 week in a
// row", "2 months in a row within the limit".
func StreakWords(n int, h Habit) string {
	h = Normal(h)
	s := fmt.Sprintf("%d %ss in a row", n, h.Cadence)
	if n == 1 {
		s = fmt.Sprintf("1 %s in a row", h.Cadence)
	}
	if h.Aim == Limit {
		s += " within the limit"
	}
	return s
}
