package when

import (
	"strings"
	"time"
)

// The words, read one at a time. Each kind of word says what it can and
// hands the rest on; a word none of them know ends the reading, unread.

// reading is what the words have said so far.
type reading struct {
	now, start, base time.Time
	haveDay          bool
	hour, minute     int
	// weekday is a day of the week written beside a date, to be checked
	// against it: Fri 19 Sep when the 19th is a Saturday is two facts that
	// disagree, and which one was meant is the person's to say.
	weekday time.Weekday
	// bad is a date or a time that does not exist, 31 Feb or 25:00: asked
	// again, not rolled over into the next month or day.
	bad bool
}

func words(s string, now time.Time) (time.Time, bool, bool) {
	r := &reading{now: now, start: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), hour: -1, weekday: -1}
	// A full stop after a word is an abbreviation, Sep. or Fri.; between
	// numbers it is part of a date or a time, 19.9.2026 or 14.30.
	w := strings.Fields(abbrevRe.ReplaceAllString(strings.ReplaceAll(s, ",", " "), "$1 "))
	for i := 0; i < len(w); i++ {
		n, ok := r.word(w, i)
		if !ok || r.bad {
			return time.Time{}, false, false
		}
		if n < 0 { // a moment was named outright: in 3 hours, now
			return r.base, false, true
		}
		i += n
	}
	if !r.haveDay && r.hour < 0 {
		return time.Time{}, false, false
	}
	if !r.haveDay {
		r.base = r.start
	}
	if r.weekday >= 0 && r.base.Weekday() != r.weekday {
		return time.Time{}, false, false
	}
	if r.hour < 0 {
		return r.base, true, true
	}
	return r.base.Add(time.Duration(r.hour)*time.Hour + time.Duration(r.minute)*time.Minute), false, true
}

// word reads one word, with the ones after it when it needs them, and says
// how many of those it used; -1 means the reading is complete.
func (r *reading) word(w []string, i int) (int, bool) {
	s := w[i]
	next := ""
	if i+1 < len(w) {
		next = w[i+1]
	}
	switch {
	case filler[s]:
		return 0, true
	case s == "now":
		r.base = r.now
		return -1, true
	case s == "today":
		r.setDay(r.start)
		return 0, true
	case s == "tomorrow":
		r.setDay(r.start.AddDate(0, 0, 1))
		return 0, true
	case s == "yesterday":
		r.setDay(r.start.AddDate(0, 0, -1))
		return 0, true
	case s == "noon":
		r.hour, r.minute = 12, 0
		return 0, true
	case s == "midnight":
		r.hour, r.minute = 0, 0
		return 0, true
	case s == "next" || s == "last":
		return r.next(s == "last", next)
	case weekdayOf(s) >= 0:
		// A weekday beside a date, as in Fri 19 Sep, must be that date's.
		r.weekday = weekdayOf(s)
		if !r.haveDay {
			r.setDay(weekdayFrom(r.start, weekdayOf(s), false))
		}
		return 0, true
	case months[s] != 0:
		return r.month(months[s], w[i+1:])
	}
	if n, ok := r.number(s, next); ok {
		return n, true
	}
	if isNumber(s) || ordinalRe.MatchString(s) {
		return r.plain(s, w[i+1:])
	}
	return 0, false
}

// number reads a clock (2pm, 14:00, 2:30pm) or a numeric date (19/9,
// 19/9/2026); a bare number is left to plain.
func (r *reading) number(s, next string) (int, bool) {
	if m := yearFirstRe.FindStringSubmatch(s); m != nil {
		r.setReal(atoi(m[1]), atoi(m[2]), atoi(m[3]))
		return 0, true
	}
	if m := numericRe.FindStringSubmatch(s); m != nil {
		d, sep, mo := atoi(m[1]), m[2], atoi(m[3])
		// 9/19 is the month first, as some write it; with dots the day
		// always comes first.
		if sep != "." && mo > 12 && d <= 12 {
			d, mo = mo, d
		}
		y := r.now.Year()
		if m[4] != "" {
			y = atoi(m[4])
			if y < 100 {
				y += 2000
			}
		}
		// 14.30 is not a date, and is how many write a time.
		if !exists(y, mo, d) && sep == "." && m[4] == "" && len(m[3]) == 2 {
			r.clock(atoi(m[1]), atoi(m[3]), "")
			return 0, true
		}
		r.setReal(y, mo, d)
		return 0, true
	}
	if m := clockRe.FindStringSubmatch(s); m != nil && (m[2] != "" || m[3] != "") {
		r.clock(atoi(m[1]), atoi(m[2]), m[3])
		return 0, true
	}
	if isNumber(s) && (next == "am" || next == "pm") {
		r.clock(atoi(s), 0, next)
		return 1, true
	}
	return 0, false
}

// plain reads a bare number: a day before a month (19 Sep), a count with a
// unit (3 days, 2 weeks ago), a day of this month (the 3rd), or an hour
// once the day is known (tomorrow at 9).
func (r *reading) plain(s string, rest []string) (int, bool) {
	if m := ordinalRe.FindStringSubmatch(s); m != nil {
		s = m[1]
	}
	n := atoi(s)
	next := ""
	if len(rest) > 0 {
		next = rest[0]
	}
	if mo := months[next]; mo != 0 {
		used := 1
		y := r.now.Year()
		if len(rest) > 1 && len(rest[1]) == 4 && isNumber(rest[1]) {
			y, used = atoi(rest[1]), 2
		}
		r.setReal(y, int(mo), n)
		return used, true
	}
	if unit := strings.TrimSuffix(next, "s"); unit == "day" || unit == "week" || unit == "month" || unit == "year" || unit == "hour" || unit == "minute" {
		used := 1
		if len(rest) > 1 && rest[1] == "ago" {
			n, used = -n, 2
		}
		switch unit {
		case "hour":
			r.base = r.now.Add(time.Duration(n) * time.Hour)
			return -1, true
		case "minute":
			r.base = r.now.Add(time.Duration(n) * time.Minute)
			return -1, true
		case "day":
			r.setDay(r.start.AddDate(0, 0, n))
		case "week":
			r.setDay(r.start.AddDate(0, 0, 7*n))
		case "month":
			r.setDay(r.start.AddDate(0, n, 0))
		case "year":
			r.setDay(r.start.AddDate(n, 0, 0))
		}
		return used, true
	}
	if !r.haveDay && n >= 1 && n <= 31 {
		// The next day of the month with that number: the 31st in
		// September is 31 October, not 1 October.
		for k := 0; k < 3; k++ {
			if d, ok := realDay(r.now.Year(), int(r.now.Month())+k, n, r.now.Location()); ok && !d.Before(r.start) {
				r.setDay(d)
				return 0, true
			}
		}
		r.bad = true
		return 0, true
	}
	if r.haveDay && n <= 23 {
		r.clock(n, 0, "")
		return 0, true
	}
	return 0, false
}

// next reads next friday, next week, next month, last monday.
func (r *reading) next(back bool, unit string) (int, bool) {
	if wd := weekdayOf(unit); wd >= 0 {
		if back {
			r.setDay(weekdayBefore(r.start, wd))
		} else {
			r.setDay(weekdayFrom(r.start, wd, true))
		}
		return 1, true
	}
	n := 1
	if back {
		n = -1
	}
	switch unit {
	case "week":
		r.setDay(r.start.AddDate(0, 0, 7*n))
	case "month":
		r.setDay(r.start.AddDate(0, n, 0))
	case "year":
		r.setDay(r.start.AddDate(n, 0, 0))
	default:
		return 0, false
	}
	return 1, true
}

// month reads a month named first: Sep 19, September 19 2026, or the
// month alone, which is its first day.
func (r *reading) month(mo time.Month, rest []string) (int, bool) {
	d, y, used := 1, r.now.Year(), 0
	if len(rest) > 0 {
		s := rest[0]
		if m := ordinalRe.FindStringSubmatch(s); m != nil {
			s = m[1]
		}
		if isNumber(s) && len(s) <= 2 {
			d, used = atoi(s), 1
			if len(rest) > 1 && len(rest[1]) == 4 && isNumber(rest[1]) {
				y, used = atoi(rest[1]), 2
			}
		}
	}
	r.setReal(y, int(mo), d)
	return used, true
}

func (r *reading) setDay(d time.Time) { r.base, r.haveDay = d, true }

// setReal sets the day when it exists, and marks the reading bad when it
// does not.
func (r *reading) setReal(y, mo, d int) {
	if !exists(y, mo, d) {
		r.bad = true
		return
	}
	day, _ := realDay(y, mo, d, r.now.Location())
	r.setDay(day)
}

// exists is whether a date written out in full is one.
func exists(y, mo, d int) bool {
	_, ok := realDay(y, mo, d, time.UTC)
	return ok && mo >= 1 && mo <= 12
}

// realDay is the date, when there is one: time.Date would take 31 Feb as
// 3 Mar, which is not what anyone wrote. A month past 12 wraps into the
// next year, as months counted on from now do.
func realDay(y, mo, d int, loc *time.Location) (time.Time, bool) {
	t := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, loc)
	return t, d >= 1 && t.Day() == d
}

func (r *reading) clock(h, m int, half string) {
	// 25:00, 13pm and 14:75 are not times.
	if h > 23 || m > 59 || (half != "" && (h < 1 || h > 12)) {
		r.bad = true
		return
	}
	if half == "pm" && h < 12 {
		h += 12
	}
	if half == "am" && h == 12 {
		h = 0
	}
	r.hour, r.minute = h, m
}
