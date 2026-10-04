// Package when reads a day or a moment the way a person writes one, and
// writes one back the way a person reads it.
//
// People know the day they mean and say it in a few words: "19 Sep",
// "next Friday", "tomorrow 2pm", "in 3 days". Government design systems
// found that letting people write the month as a word instead of a
// number cut errors dramatically; usability research finds a calendar
// picker is for finding a day one does not know, and typing is faster for
// one they do. So the words come first here, every common way of writing
// them is read, and the machine forms (2026-09-19, RFC 3339) are read too,
// so an agent and a person write the same field.
package when

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Hint says what can be written, for a form field's help text.
const Hint = "A day, like 19 Sep or next Friday, with a time if there is one, like 2pm."

// Text is a stored value as a person reads it: "Sat 19 Sep 2026" for a
// day, "Sat 19 Sep 2026, 14:00" for a moment, in local time. Anything that
// is not a stored value comes back as it is.
func Text(v string) string {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return v
	}
	if strings.HasSuffix(v, "T00:00:00Z") {
		return ts.UTC().Format("Mon 2 Jan 2006")
	}
	return ts.Local().Format("Mon 2 Jan 2006, 15:04")
}

// Store is a parsed value as it is kept: a day as midnight UTC on that
// date, which is the same day everywhere, and a moment in UTC.
func Store(t time.Time, day bool) string {
	if day {
		return t.Format("2006-01-02") + "T00:00:00Z"
	}
	return t.UTC().Format(time.RFC3339)
}

var (
	clockRe     = regexp.MustCompile(`^(\d{1,2})(?:[:.](\d{2}))?(am|pm)?$`)
	numericRe   = regexp.MustCompile(`^(\d{1,2})([/.-])(\d{1,2})(?:[/.-](\d{2,4}))?$`)
	yearFirstRe = regexp.MustCompile(`^(\d{4})[/.](\d{1,2})[/.](\d{1,2})$`)
	abbrevRe    = regexp.MustCompile(`([a-z])\.`)
	ordinalRe   = regexp.MustCompile(`^(\d{1,2})(st|nd|rd|th)$`)
	weekdays    = map[string]time.Weekday{"mon": 1, "monday": 1, "tue": 2, "tues": 2, "tuesday": 2, "wed": 3, "wednesday": 3, "thu": 4, "thur": 4, "thurs": 4, "thursday": 4, "fri": 5, "friday": 5, "sat": 6, "saturday": 6, "sun": 0, "sunday": 0}
	months      = map[string]time.Month{"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3, "apr": 4, "april": 4, "may": 5, "jun": 6, "june": 6, "jul": 7, "july": 7, "aug": 8, "august": 8, "sep": 9, "sept": 9, "september": 9, "oct": 10, "october": 10, "nov": 11, "november": 11, "dec": 12, "december": 12}
	filler      = map[string]bool{"at": true, "on": true, "the": true, "of": true, "in": true, "this": true, "and": true}
)

// Parse reads s as a day or a moment, relative to now for words like
// today. day says s named a whole day rather than a time within one. ok is
// false when a word was not understood: better to ask again than to guess.
func Parse(s string, now time.Time) (t time.Time, day bool, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false, false
	}
	if t, err := time.Parse(time.RFC3339, strings.ToUpper(s)); err == nil {
		return t, false, true
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, s, now.Location()); err == nil {
			return t, false, true
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", s, now.Location()); err == nil {
		return t, true, true
	}
	// 2026-10-06 2pm: a date with its time in words, as models write it.
	if len(s) > 11 && s[10] == ' ' {
		if d, err := time.ParseInLocation("2006-01-02", s[:10], now.Location()); err == nil {
			if t, day, ok := words("today "+strings.ToLower(s[11:]), d); ok && !day && t.Format("2006-01-02") == s[:10] {
				return t, false, true
			}
		}
	}
	if t, ok := shift(s, now); ok {
		return t, false, true
	}
	return words(strings.ToLower(s), now)
}

// shift is +3d, -1w, +3h: a distance from now.
func shift(s string, now time.Time) (time.Time, bool) {
	if len(s) < 3 || (s[0] != '+' && s[0] != '-') {
		return time.Time{}, false
	}
	n, err := strconv.Atoi(s[1 : len(s)-1])
	if err != nil {
		return time.Time{}, false
	}
	if s[0] == '-' {
		n = -n
	}
	switch s[len(s)-1] {
	case 'd':
		return now.AddDate(0, 0, n), true
	case 'w':
		return now.AddDate(0, 0, 7*n), true
	case 'h':
		return now.Add(time.Duration(n) * time.Hour), true
	}
	return time.Time{}, false
}

func weekdayOf(s string) time.Weekday {
	if wd, ok := weekdays[s]; ok {
		return wd
	}
	return -1
}

// weekdayFrom is the next day with that weekday, today included unless
// strict, which is what "next Friday" means when today is Friday.
func weekdayFrom(start time.Time, wd time.Weekday, strict bool) time.Time {
	diff := (int(wd) - int(start.Weekday()) + 7) % 7
	if diff == 0 && strict {
		diff = 7
	}
	return start.AddDate(0, 0, diff)
}

func weekdayBefore(start time.Time, wd time.Weekday) time.Time {
	diff := (int(start.Weekday()) - int(wd) + 7) % 7
	if diff == 0 {
		diff = 7
	}
	return start.AddDate(0, 0, -diff)
}

func isNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil && s != ""
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// Short is a stored value the way a list says it at the right of a row:
// Today, Tomorrow, the weekday within the week, else the day and month,
// with the year only when it is another year, and the time when there is
// one. Anything that is not a stored value comes back as it is.
func Short(v string, now time.Time) string {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return v
	}
	day, clock := ts.UTC(), ""
	if !strings.HasSuffix(v, "T00:00:00Z") {
		day, clock = ts.Local(), " "+ts.Local().Format("15:04")
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, now.Location())
	diff := int(d.Sub(today).Hours() / 24)
	switch {
	case diff == 0:
		return "Today" + clock
	case diff == 1:
		return "Tomorrow" + clock
	case diff > 1 && diff < 7:
		return d.Format("Monday") + clock
	case diff < 0 && diff > -7:
		return d.Format("Mon 2 Jan") + clock
	case d.Year() == now.Year():
		return d.Format("2 Jan") + clock
	}
	return d.Format("2 Jan 2006") + clock
}

// Relative is a stored value the way a person reads it on a list row:
// "Today at 12:53am", "Yesterday at 6pm", "Two days ago at 9pm", or
// "Last Monday" for more than a week back. The time uses a 12-hour clock
// with am/pm; dates far away omit the time.
// Anything that is not a stored value comes back as it is.
func Relative(v string, now time.Time) string {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return v
	}

	dayOnly := strings.HasSuffix(v, "T00:00:00Z")
	localDay := ts.Local()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	d := time.Date(localDay.Year(), localDay.Month(), localDay.Day(), 0, 0, 0, 0, now.Location())
	diff := int(d.Sub(today).Hours() / 24)

	var datePart string
	switch {
	case diff == 0:
		datePart = "Today"
	case diff == 1:
		datePart = "Tomorrow"
	case diff > 1 && diff < 7:
		datePart = d.Weekday().String() + " at " + timeStr(localDay)
		return datePart
	case diff < 0 && diff > -7:
		n := -diff
		if n == 1 {
			datePart = "Yesterday"
		} else {
			datePart = fmt.Sprintf("%d days ago", n)
		}
	default:
		// More than a week back — use the month/day form.
		datePart = localDay.Format("2 Jan")
		if !dayOnly {
			return datePart + " at " + timeStr(localDay)
		}
		return datePart
	}

	if dayOnly {
		return datePart
	}
	return datePart + " at " + timeStr(localDay)
}

// timeStr formats a time as 12-hour with am/pm, like "1:53am" or "6pm".
func timeStr(t time.Time) string {
	h := t.Hour()
	m := t.Minute()
	suffix := "am"
	switch {
	case h == 0:
		h = 12
	case h == 12:
		suffix = "pm"
	case h > 12:
		h -= 12
		suffix = "pm"
	}
	if m == 0 {
		return fmt.Sprintf("%d%s", h, suffix)
	}
	return fmt.Sprintf("%d:%02d%s", h, m, suffix)
}

// setReal sets the day when it exists, and marks the reading bad when it
// does not.
func (r *reading) setReal(y, mo, d int) {
	if !exists(y, mo, d) {
		r.bad, r.why = true, "day"
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

// Why says why words are not read as a day or a moment, as what they must
// be: a real day, a real time, one day and not two that disagree, or,
// failing those, a day at all. Nothing when they are read.
func Why(s string, now time.Time) string {
	if _, _, ok := Parse(s, now); ok {
		return ""
	}
	r := newReading(now)
	r.read(strings.ToLower(strings.TrimSpace(s)))
	said := strings.TrimSpace(s)
	switch r.why {
	case "day":
		return "must be a real day: " + said + " is not one"
	case "time":
		return "must be a real time: " + said + " is not one"
	case "weekday":
		return "must be one day: " + r.base.Format("2 Jan 2006") + " is a " + r.base.Weekday().String() + ", not a " + r.weekday.String()
	}
	return "must be a day, like 19 Sep, next Friday or tomorrow 2pm"
}
