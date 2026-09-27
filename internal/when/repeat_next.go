package when

import (
	"strings"
	"time"
)

// When a repeat falls. The schedule is counted on from the day something
// was due, not from when it was done: "every Tuesday" stays on Tuesdays
// however late the last one was ticked, which is what a calendar and
// Apple's Reminders do. A day that some months lack, the 31st or 29 Feb,
// falls on the last day those months have, since a person who wrote the
// 31st meant every month, not every month that has one (RFC 5545 would
// skip them). A time of day is kept as the clock on the wall says it,
// across a change to or from summer time, as RFC 5545 asks.

// Next is when a repeat falls next after the time in v and after now, as
// it is stored: a day stays a day, a moment keeps its time. Something
// overdue moves to the next time on its schedule that is still to come,
// not to one already past. With nothing in v it counts from today. ok is
// false when the repeat has ended, or is not one.
func Next(repeat, v string, now time.Time) (string, bool) {
	r, ok := stored(repeat)
	if !ok {
		return "", false
	}
	base, loc, day := start(v, now)
	after := base
	if floor := today(now, loc, day); floor.After(after) {
		after = floor
	}
	var next time.Time
	r.each(base, loc, func(t time.Time) bool {
		if t.After(after) {
			next = t
			return false
		}
		return true
	})
	if next.IsZero() {
		return "", false
	}
	return Store(next, day), true
}

// Pin fixes the day a monthly or yearly repeat falls on to the day in v,
// when it names none: "every month" from the 31st is every month on the
// 31st, so that after February's 28th it goes back to the 31st rather
// than keeping to the 28th. Anything else comes back as it is.
func Pin(repeat, v string) string {
	r, ok := stored(repeat)
	ts, err := time.Parse(time.RFC3339, v)
	if !ok || err != nil || len(r.days) > 0 || r.monthDay != 0 {
		return repeat
	}
	if !strings.HasSuffix(v, "T00:00:00Z") {
		ts = ts.Local()
	}
	switch r.freq {
	case "MONTHLY":
		r.monthDay = ts.Day()
	case "YEARLY":
		r.month, r.monthDay = ts.Month(), ts.Day()
	default:
		return repeat
	}
	return r.String()
}

// KeepTime holds a repeat to the time of day in v, a moment, while that
// one time is put off: an alarm every day at 07:00 given five more minutes
// still rings at 07:00 tomorrow, not 07:05. DropTime lets it go again once
// the next time is set, so the time in the field is the one kept.
func KeepTime(repeat, v string) string {
	r, ok := stored(repeat)
	ts, err := time.Parse(time.RFC3339, v)
	if !ok || err != nil || r.clock >= 0 || strings.HasSuffix(v, "T00:00:00Z") {
		return repeat
	}
	ts = ts.Local()
	r.clock = ts.Hour()*60 + ts.Minute()
	return r.String()
}

// DropTime is the repeat without a time held by KeepTime.
func DropTime(repeat string) string {
	r, ok := stored(repeat)
	if !ok || r.clock < 0 {
		return repeat
	}
	r.clock = -1
	return r.String()
}

// start is the time the schedule counts from, the place its days are in,
// and whether it is a whole day: a day is kept in UTC, a moment is read
// in local time.
func start(v string, now time.Time) (time.Time, *time.Location, bool) {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return today(now, time.UTC, true), time.UTC, true
	}
	if strings.HasSuffix(v, "T00:00:00Z") {
		return ts.UTC(), time.UTC, true
	}
	return ts.In(now.Location()), now.Location(), false
}

// today is now, or for a whole day, today's date where the person is.
func today(now time.Time, loc *time.Location, day bool) time.Time {
	if !day {
		return now
	}
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
}

// each gives every time the repeat falls, in order, counted from base,
// until fn says stop or the repeat ends. It may begin before base.
func (r rule) each(base time.Time, loc *time.Location, fn func(time.Time) bool) {
	h, m, s := base.Clock()
	if r.clock >= 0 {
		h, m, s = r.clock/60, r.clock%60, 0
	}
	y, mo, d := base.Date()
	at := func(y int, mo time.Month, d int) time.Time { return time.Date(y, mo, d, h, m, s, 0, loc) }
	// A bound, so a rule that never falls cannot loop for ever: a daily
	// repeat for well over a century.
	for k, off := 0, 0; k < 50000; k++ {
		var t time.Time
		switch {
		case r.freq == "DAILY":
			t = at(y, mo, d+k*r.every)
		case r.freq == "WEEKLY" && len(r.days) == 0:
			t = at(y, mo, d+7*k*r.every)
		case r.freq == "WEEKLY" && r.every == 1:
			t = at(y, mo, d+k)
			if !hasDay(r.days, t.Weekday()) {
				continue
			}
		case r.freq == "WEEKLY":
			if k == 0 {
				off = (int(r.days[0]) - int(base.Weekday()) + 7) % 7
			}
			t = at(y, mo, d+off+7*k*r.every)
		case r.freq == "MONTHLY":
			first := time.Date(y, mo+time.Month(k*r.every), 1, 0, 0, 0, 0, loc)
			t = at(first.Year(), first.Month(), fit(r.monthDay, d, first.Year(), first.Month()))
		case r.freq == "YEARLY":
			month := mo
			if r.month != 0 {
				month = r.month
			}
			t = at(y+k*r.every, month, fit(r.monthDay, d, y+k*r.every, month))
		}
		if !r.until.IsZero() && dateOf(t).After(r.until) {
			return
		}
		if !fn(t) {
			return
		}
	}
}

// fit is the day of the month a repeat falls on in that month: the day
// asked for (or the one it started on), or the month's last day when it
// has fewer.
func fit(want, started, y int, mo time.Month) int {
	if want == 0 {
		want = started
	}
	last := time.Date(y, mo+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if want == -1 || want > last {
		return last
	}
	return want
}

func hasDay(days []time.Weekday, wd time.Weekday) bool {
	for _, d := range days {
		if d == wd {
			return true
		}
	}
	return false
}

// dateOf is the date of t, where t is, as midnight UTC.
func dateOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
