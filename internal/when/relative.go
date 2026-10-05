package when

import (
	"fmt"
	"strings"
	"time"
)

// How a day or a moment is said to a person, everywhere: one set of words,
// so a task is not "In 4 days at 2:00pm" in its list, "Fri 9 Oct 2026,
// 14:00" in its fields and "Due Friday" to the assistant. The rules are
// design/foundations/glance.md's:
//
//   - A day planned for (due, starts, rings) is said as a day a person
//     plans by: Today, Tomorrow, Yesterday, else its weekday and date,
//     "Fri 9 Oct", with the year only when it is not this one. "In 4
//     days" makes a person count, and goes wrong on a page left open.
//   - When something happened (made, changed) is said as how long ago,
//     while that is short: "Today at 2pm", "3 days ago", then its date.
//   - A time of day is the person's clock (Clock): "2pm", "5:30pm",
//     "midday", "midnight"; or "14:00" on the 24-hour clock.
//   - Day, month, year, in that order, the month a word, no commas.

// Hours24 says whether times of day are said on the 24-hour clock. The
// app sets it from the person's setting (ui.clock) and language; unset,
// the 12-hour clock is used.
var Hours24 func() bool

// TwentyFour is whether a person who chose clock ("12", "24" or nothing)
// and writes in language reads times on the 24-hour clock: what they
// chose, else what their language uses (CLDR's hour cycles). English is
// said the GOV.UK way, 5:30pm, whatever the region.
func TwentyFour(clock, language string) bool {
	switch clock {
	case "12":
		return false
	case "24":
		return true
	}
	lang := strings.ToLower(strings.SplitN(strings.ReplaceAll(language, "_", "-"), "-", 2)[0])
	return !twelve[lang]
}

// twelve are the languages whose clock is the 12-hour one.
var twelve = map[string]bool{"": true, "en": true, "hi": true, "bn": true, "ur": true, "ar": true, "ko": true, "fil": true, "tl": true, "ml": true, "ta": true, "te": true, "mr": true, "gu": true, "pa": true}

func use24() bool { return Hours24 != nil && Hours24() }

// Clock is a time of day as a person says it: "2pm", "5:30pm", "midday",
// "midnight" (GOV.UK's style: never 12pm, never 14:00hrs), or "14:00" and
// "09:30" on the 24-hour clock.
func Clock(t time.Time) string {
	h, m := t.Hour(), t.Minute()
	if use24() {
		return fmt.Sprintf("%02d:%02d", h, m)
	}
	switch {
	case h == 0 && m == 0:
		return "midnight"
	case h == 12 && m == 0:
		return "midday"
	}
	half := "am"
	if h >= 12 {
		half = "pm"
	}
	if h = h % 12; h == 0 {
		h = 12
	}
	if m == 0 {
		return fmt.Sprintf("%d%s", h, half)
	}
	return fmt.Sprintf("%d:%02d%s", h, m, half)
}

// Face is the time on a clock's face, always with its minutes: "2:05pm",
// "2:00pm" or "14:05". A face ticks, and "2pm" then "2:01pm" would jump.
func Face(t time.Time) string {
	if use24() {
		return t.Format("15:04")
	}
	return strings.ToLower(t.Format("3:04pm"))
}

// Relative is a stored day or moment planned for, as a person plans by it
// (see above): "Today", "Tomorrow at 2pm", "Fri 9 Oct at 5:30pm",
// "Mon 4 Jan 2027". A moment is said in the reader's zone, the one now
// is in. Anything that is not a stored value comes back as it is.
func Relative(v string, now time.Time) string {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return v
	}
	if IsDay(v) {
		return dayWords(ts.UTC(), now)
	}
	return At(ts, now)
}

// At is a moment planned for, in the reader's zone: "Tomorrow at 2pm".
func At(t, now time.Time) string {
	local := t.In(now.Location())
	return dayWords(local, now) + " at " + Clock(local)
}

// Day is a date as a person plans by it: Today, Tomorrow, Yesterday, or
// its weekday and date, with the year when it is not this one.
func Day(d, now time.Time) string { return dayWords(d, now) }

// IsDay is whether a stored value is a whole day rather than a moment.
func IsDay(v string) bool { return strings.HasSuffix(v, "T00:00:00Z") }

func dayWords(d, now time.Time) string {
	switch daysFrom(d, now) {
	case 0:
		return "Today"
	case 1:
		return "Tomorrow"
	case -1:
		return "Yesterday"
	}
	if d.Year() == now.Year() {
		return d.Format("Mon 2 Jan")
	}
	return d.Format("Mon 2 Jan 2006")
}

// daysFrom is how many days d's date is after now's, by the calendar.
func daysFrom(d, now time.Time) int {
	a := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return int(a.Sub(b).Hours() / 24)
}

// Ago is when something happened, said by how long ago while that is
// short: "Today at 2pm", "Yesterday at 9:30am", "3 days ago", then its
// date, "2 Oct", "2 Oct 2025". A time from a clock a little ahead is
// today, not tomorrow.
func Ago(t, now time.Time) string {
	local := t.In(now.Location())
	n := daysFrom(local, now)
	switch {
	case n >= 0:
		return "Today at " + Clock(local)
	case n == -1:
		return "Yesterday at " + Clock(local)
	case n > -7:
		return fmt.Sprintf("%d days ago", -n)
	case local.Year() == now.Year():
		return local.Format("2 Jan")
	}
	return local.Format("2 Jan 2006")
}

// Full is a stored day or moment in full, for when the words above leave
// the date out (Today, 3 days ago): "Monday 5 October 2026 at 2pm".
func Full(v string) string {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return v
	}
	if IsDay(v) {
		return ts.UTC().Format("Monday 2 January 2006")
	}
	return ts.Local().Format("Monday 2 January 2006") + " at " + Clock(ts.Local())
}

// Machine is a stored value as a <time datetime> holds it: the date alone
// for a day, the moment with its offset otherwise.
func Machine(v string) string {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return ""
	}
	if IsDay(v) {
		return ts.UTC().Format("2006-01-02")
	}
	return ts.Format(time.RFC3339)
}

// LeavesDateOut is whether words leave the date out (Today, 3 days
// ago), so the date in full is given beside them for a pointer.
func LeavesDateOut(words string) bool {
	for _, w := range []string{"Today", "Tomorrow", "Yesterday"} {
		if strings.HasPrefix(words, w) {
			return true
		}
	}
	return strings.Contains(words, " ago")
}
