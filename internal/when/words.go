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
}

func words(s string, now time.Time) (time.Time, bool, bool) {
	r := &reading{now: now, start: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), hour: -1}
	w := strings.Fields(strings.NewReplacer(",", " ", ".", " ").Replace(s))
	for i := 0; i < len(w); i++ {
		n, ok := r.word(w, i)
		if !ok {
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
		// A weekday before an explicit date, as in Fri 19 Sep, is only a
		// reminder; the date wins.
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
	if m := numericRe.FindStringSubmatch(s); m != nil {
		d, mo := atoi(m[1]), atoi(m[2])
		if mo > 12 && d <= 12 {
			d, mo = mo, d
		}
		y := r.now.Year()
		if m[3] != "" {
			y = atoi(m[3])
			if y < 100 {
				y += 2000
			}
		}
		r.setDay(time.Date(y, time.Month(mo), d, 0, 0, 0, 0, r.now.Location()))
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
		r.setDay(time.Date(y, mo, n, 0, 0, 0, 0, r.now.Location()))
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
		d := time.Date(r.now.Year(), r.now.Month(), n, 0, 0, 0, 0, r.now.Location())
		if d.Before(r.start) {
			d = d.AddDate(0, 1, 0)
		}
		r.setDay(d)
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
	r.setDay(time.Date(y, mo, d, 0, 0, 0, 0, r.now.Location()))
	return used, true
}

func (r *reading) setDay(d time.Time) { r.base, r.haveDay = d, true }

func (r *reading) clock(h, m int, half string) {
	if half == "pm" && h < 12 {
		h += 12
	}
	if half == "am" && h == 12 {
		h = 0
	}
	r.hour, r.minute = h, m
}
