package when

import (
	"strings"
	"time"
)

// Months, days over a list, a short date beside a name and when a message
// was sent were each said by the page that showed them, in four copies of
// "January 2006" and three of "2 Jan"; one chat's list left the year off a
// chat from last year. They are said here, with the rest of how a day is
// said (design/foundations/glance.md).

// Month is a month, kept as 2026-09, as a heading says it: "September
// 2026". Anything else is given back as it came.
func Month(v string) string {
	t, err := time.Parse("2006-01", strings.TrimSpace(v))
	if err != nil {
		return v
	}
	return t.Format("January 2006")
}

// DayHeading is a day as the heading over what happened on it: "Today,
// Monday 5 October", "Yesterday, ...", the weekday and date, and the year
// when it is not this one.
func DayHeading(at, now time.Time) string {
	at = at.In(now.Location())
	switch daysFrom(at, now) {
	case 0:
		return "Today, " + at.Format("Monday 2 January")
	case -1:
		return "Yesterday, " + at.Format("Monday 2 January")
	}
	if at.Year() != now.Year() {
		return at.Format("Monday 2 January 2006")
	}
	return at.Format("Monday 2 January")
}

// Date is a day short, beside a name: "2 Oct", with the year when it is
// not this one, "2 Oct 2025".
func Date(at, now time.Time) string {
	at = at.In(now.Location())
	if at.Year() == now.Year() {
		return at.Format("2 Jan")
	}
	return at.Format("2 Jan 2006")
}

// Sent is when a message was sent, as its chat shows it: the time alone
// today, its Date and time before, so an old chat reads true.
func Sent(at, now time.Time) string {
	at = at.In(now.Location())
	if daysFrom(at, now) == 0 {
		return Clock(at)
	}
	return Date(at, now) + " at " + Clock(at)
}
