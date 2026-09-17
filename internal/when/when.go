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
	clockRe   = regexp.MustCompile(`^(\d{1,2})(?::(\d{2}))?(am|pm)?$`)
	numericRe = regexp.MustCompile(`^(\d{1,2})[/-](\d{1,2})(?:[/-](\d{2,4}))?$`)
	ordinalRe = regexp.MustCompile(`^(\d{1,2})(st|nd|rd|th)$`)
	weekdays  = map[string]time.Weekday{"mon": 1, "monday": 1, "tue": 2, "tues": 2, "tuesday": 2, "wed": 3, "wednesday": 3, "thu": 4, "thur": 4, "thurs": 4, "thursday": 4, "fri": 5, "friday": 5, "sat": 6, "saturday": 6, "sun": 0, "sunday": 0}
	months    = map[string]time.Month{"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3, "apr": 4, "april": 4, "may": 5, "jun": 6, "june": 6, "jul": 7, "july": 7, "aug": 8, "august": 8, "sep": 9, "sept": 9, "september": 9, "oct": 10, "october": 10, "nov": 11, "november": 11, "dec": 12, "december": 12}
	filler    = map[string]bool{"at": true, "on": true, "the": true, "of": true, "in": true, "this": true, "and": true}
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
