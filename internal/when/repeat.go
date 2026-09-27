package when

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// How often something happens again, as a person says it: "every day",
// "every Tuesday", "every 2 weeks", "every month on the 1st", "until 1
// Mar". People already say a repeat in a few words, and the calendars and
// to-do lists they know read it from them (Todoist, Fantastical, Google
// Calendar's quick add), where a picker of dropdowns makes the same rule
// five choices long. It is kept as a small part of iCalendar's RRULE (RFC
// 5545), the form every calendar exchanges, and said back in words, so a
// wrong reading is caught before it matters.

// RepeatHint says what can be written, for a form field's help text.
const RepeatHint = "How often, like every day, every Tuesday, every 2 weeks or every month on the 1st; add until 1 Mar if it ends. Empty for once."

// rule is a repeat as it is kept: how often (freq and every), on which
// weekdays, on which day of the month (-1 is the last) and in which month,
// and the last day it may fall on.
type rule struct {
	freq     string // DAILY, WEEKLY, MONTHLY or YEARLY
	every    int
	days     []time.Weekday
	monthDay int
	month    time.Month
	until    time.Time // a day, midnight UTC; zero when it goes on
	// clock is the time of day it falls at, as hours and minutes, kept
	// only while one time is put off (KeepTime); -1 when the time in the
	// field is the time.
	clock int
}

var rruleDays = []string{"SU", "MO", "TU", "WE", "TH", "FR", "SA"}

// String is the rule as it is stored, the RRULE parts in one order:
// FREQ=WEEKLY;INTERVAL=2;BYDAY=TU;UNTIL=20270301.
func (r rule) String() string {
	parts := []string{"FREQ=" + r.freq}
	if r.every > 1 {
		parts = append(parts, "INTERVAL="+strconv.Itoa(r.every))
	}
	if len(r.days) > 0 {
		var d []string
		for _, wd := range r.days {
			d = append(d, rruleDays[wd])
		}
		parts = append(parts, "BYDAY="+strings.Join(d, ","))
	}
	if r.month != 0 {
		parts = append(parts, "BYMONTH="+strconv.Itoa(int(r.month)))
	}
	if r.monthDay != 0 {
		parts = append(parts, "BYMONTHDAY="+strconv.Itoa(r.monthDay))
	}
	if r.clock >= 0 {
		parts = append(parts, "BYHOUR="+strconv.Itoa(r.clock/60), "BYMINUTE="+strconv.Itoa(r.clock%60))
	}
	if !r.until.IsZero() {
		parts = append(parts, "UNTIL="+r.until.Format("20060102"))
	}
	return strings.Join(parts, ";")
}

// stored reads a rule as String writes it; ok is false for anything else,
// including the RRULE parts this does not keep (BYSETPOS, COUNT and the
// like), which are refused rather than half obeyed.
func stored(s string) (rule, bool) {
	r := rule{every: 1, clock: -1}
	hour, minute := -1, 0
	if !strings.HasPrefix(strings.ToUpper(s), "FREQ=") {
		return r, false
	}
	for _, part := range strings.Split(strings.ToUpper(strings.TrimSpace(s)), ";") {
		k, v, _ := strings.Cut(part, "=")
		n, err := strconv.Atoi(v)
		switch k {
		case "FREQ":
			if v != "DAILY" && v != "WEEKLY" && v != "MONTHLY" && v != "YEARLY" {
				return r, false
			}
			r.freq = v
		case "INTERVAL":
			if err != nil || n < 1 || n > 999 {
				return r, false
			}
			r.every = n
		case "BYDAY":
			for _, d := range strings.Split(v, ",") {
				i := indexOf(rruleDays, d)
				if i < 0 {
					return r, false
				}
				r.days = append(r.days, time.Weekday(i))
			}
		case "BYMONTH":
			if err != nil || n < 1 || n > 12 {
				return r, false
			}
			r.month = time.Month(n)
		case "BYMONTHDAY":
			if err != nil || n < -1 || n > 31 || n == 0 {
				return r, false
			}
			r.monthDay = n
		case "BYHOUR":
			if err != nil || n < 0 || n > 23 {
				return r, false
			}
			hour = n
		case "BYMINUTE":
			if err != nil || n < 0 || n > 59 {
				return r, false
			}
			minute = n
		case "UNTIL":
			u, err := time.Parse("20060102", v[:min(8, len(v))])
			if err != nil {
				return r, false
			}
			r.until = u
		default:
			return r, false
		}
	}
	if hour >= 0 {
		r.clock = hour*60 + minute
	}
	return r, r.freq != ""
}

func indexOf(list []string, s string) int {
	for i, x := range list {
		if x == s {
			return i
		}
	}
	return -1
}

// RepeatText is a stored repeat as a person reads it, "every Tuesday until
// Mon 1 Mar 2027", in words that read back as the same repeat. Anything
// that is not a stored repeat comes back as it is.
func RepeatText(v string) string {
	r, ok := stored(v)
	if !ok {
		return v
	}
	unit := map[string]string{"DAILY": "day", "WEEKLY": "week", "MONTHLY": "month", "YEARLY": "year"}[r.freq]
	s := "every " + unit
	if r.every > 1 {
		s = fmt.Sprintf("every %d %ss", r.every, unit)
	}
	switch {
	case len(r.days) > 0 && r.every == 1:
		s = "every " + dayList(r.days)
	case len(r.days) > 0:
		s += " on " + dayList(r.days)
	case r.freq == "YEARLY" && r.month != 0 && r.monthDay > 0:
		s += fmt.Sprintf(" on %d %s", r.monthDay, r.month.String()[:3])
	case r.monthDay == -1:
		s += " on the last day"
	case r.monthDay > 0:
		s += " on the " + ordinal(r.monthDay)
	}
	if !r.until.IsZero() {
		s += " until " + r.until.Format("Mon 2 Jan 2006")
	}
	return s
}

// dayList is weekdays in words: Monday to Friday is "weekday", else the
// days named, "Monday, Wednesday and Friday".
func dayList(days []time.Weekday) string {
	if len(days) == 5 && sameDays(days, weekdaysOnly) {
		return "weekday"
	}
	var names []string
	for _, d := range days {
		names = append(names, d.String())
	}
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

var weekdaysOnly = []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}

func sameDays(a, b []time.Weekday) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func ordinal(n int) string {
	suffix := "th"
	switch {
	case n%100 >= 11 && n%100 <= 13:
	case n%10 == 1:
		suffix = "st"
	case n%10 == 2:
		suffix = "nd"
	case n%10 == 3:
		suffix = "rd"
	}
	return strconv.Itoa(n) + suffix
}
