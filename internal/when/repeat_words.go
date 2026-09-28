package when

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

// A repeat read from words. Like a day, every common way of saying one is
// read, the stored form too, so an agent and a person write the same
// field; what is not understood is refused with what it must be, not
// guessed at.

// ParseRepeat reads s as a repeat and says it as it is stored. Empty
// words, or never, are no repeat: "" and ok. ok is false when the words
// were not understood.
func ParseRepeat(s string, now time.Time) (string, bool) {
	r, none, why := readRepeat(s, now)
	if why != "" {
		return "", false
	}
	if none {
		return "", true
	}
	return r.String(), true
}

// RepeatWhy says why words are not read as a repeat, as what they must
// be. Nothing when they are read.
func RepeatWhy(s string, now time.Time) string {
	_, _, why := readRepeat(s, now)
	return why
}

const repeatMust = "must say how often, like every day, every Tuesday, every 2 weeks or every month on the 1st"

var (
	noRepeat  = map[string]bool{"never": true, "once": true, "no": true, "none": true, "not": true, "does not repeat": true, "no repeat": true}
	yearRe    = regexp.MustCompile(`\d{4}|/\d{2}$`)
	shorthand = map[string][]string{"daily": {"every", "day"}, "weekly": {"every", "week"}, "fortnightly": {"every", "2", "weeks"}, "monthly": {"every", "month"}, "yearly": {"every", "year"}, "annually": {"every", "year"}}
)

// readRepeat is the reading: the rule, or none, or why the words are not
// one.
func readRepeat(s string, now time.Time) (r rule, none bool, why string) {
	r.clock = -1
	s = strings.ToLower(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), ".")))
	if s == "" || noRepeat[s] {
		return r, true, ""
	}
	if r, ok := stored(s); ok {
		return r, false, ""
	}
	w := strings.Fields(strings.ReplaceAll(s, ",", " "))
	if len(w) > 0 && (w[0] == "repeats" || w[0] == "repeat") {
		w = w[1:]
	}
	// until 1 Mar: the rest is a day, read as any day is.
	for i, x := range w {
		if x == "until" || x == "till" {
			words := strings.Join(w[i+1:], " ")
			t, _, ok := Parse(words, now)
			if !ok {
				return r, false, "must end on a real day, like until 1 Mar: " + words + " is not one"
			}
			r.until = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			// Until 1 Mar, said in September, is next March: a repeat
			// does not end before it begins. A year written is kept.
			if r.until.Before(dateOf(now)) && !yearRe.MatchString(words) {
				r.until = r.until.AddDate(1, 0, 0)
			}
			w = w[:i]
			break
		}
	}
	if len(w) > 0 && shorthand[w[0]] != nil {
		w = append(append([]string{}, shorthand[w[0]]...), w[1:]...)
	}
	if len(w) < 2 || (w[0] != "every" && w[0] != "each") {
		return r, false, repeatMust
	}
	r.every, w = 1, w[1:]
	switch {
	case isNumber(w[0]) && len(w) > 1:
		r.every, w = atoi(w[0]), w[1:]
	case w[0] == "other" && len(w) > 1:
		r.every, w = 2, w[1:]
	}
	if r.every < 1 || r.every > 999 {
		return r, false, repeatMust
	}
	why = r.unit(strings.TrimSuffix(w[0], "s"), facts(w[1:]))
	return r, false, why
}

// facts is what follows the unit, without the small words around the
// facts: on the 1st, on Tuesday and Friday, on the last day.
func facts(w []string) []string {
	var out []string
	for _, x := range w {
		if x != "on" && x != "the" && x != "of" && x != "and" && x != "day" && x != "every" {
			out = append(out, x)
		}
	}
	return out
}

// unit reads the unit and what is said after it, and says why they are
// not a repeat, if they are not.
func (r *rule) unit(unit string, rest []string) string {
	switch unit {
	case "day":
		r.freq = "DAILY"
		if len(rest) > 0 {
			return repeatMust
		}
		return ""
	case "week":
		r.freq = "WEEKLY"
		return r.weekdays(rest)
	case "weekday", "weekend":
		r.freq = "WEEKLY"
		r.days = weekdaysOnly
		if unit == "weekend" {
			r.days = []time.Weekday{time.Saturday, time.Sunday}
		}
		if r.every > 1 || len(rest) > 0 {
			return repeatMust
		}
		return ""
	case "month":
		r.freq = "MONTHLY"
		return r.dayOfMonth(rest)
	case "year":
		r.freq = "YEARLY"
		return r.dayOfYear(rest)
	}
	// every 1st, every 15th of the month, every last day of the month
	if (ordinalRe.MatchString(unit) || unit == "last") && (len(rest) == 0 || len(rest) == 1 && rest[0] == "month") && r.every == 1 {
		r.freq = "MONTHLY"
		return r.dayOfMonth([]string{unit})
	}
	r.freq = "WEEKLY"
	return r.weekdays(append([]string{unit}, rest...))
}

// weekdays reads Tuesday, or Monday and Thursday, Monday first.
func (r *rule) weekdays(names []string) string {
	seen := map[time.Weekday]bool{}
	for _, n := range names {
		wd := weekdayOf(strings.TrimSuffix(n, "s"))
		if wd < 0 {
			wd = weekdayOf(n)
		}
		if wd < 0 {
			return repeatMust
		}
		if !seen[wd] {
			seen[wd] = true
			r.days = append(r.days, wd)
		}
	}
	sort.Slice(r.days, func(i, j int) bool { return (r.days[i]+6)%7 < (r.days[j]+6)%7 })
	if r.every > 1 && len(r.days) > 1 {
		return "must fall on one day of the week when it is every few weeks, like every 2 weeks on Tuesday"
	}
	return ""
}

// dayOfMonth reads the 1st, 15, or the last (day).
func (r *rule) dayOfMonth(rest []string) string {
	switch {
	case len(rest) == 0:
		return ""
	case len(rest) == 1 && rest[0] == "last":
		r.monthDay = -1
		return ""
	case len(rest) == 1:
		n, ok := dayNumber(rest[0])
		if !ok {
			return repeatMust
		}
		if n < 1 || n > 31 {
			return "must be on a day of the month from the 1st to the 31st, or the last day"
		}
		r.monthDay = n
		return ""
	}
	return repeatMust
}

// dayOfYear reads 29 Feb or Feb 29: a day that is in the calendar, in a
// leap year at least.
func (r *rule) dayOfYear(rest []string) string {
	if len(rest) == 0 {
		return ""
	}
	if len(rest) != 2 {
		return repeatMust
	}
	day, mon := rest[0], rest[1]
	if months[day] != 0 {
		day, mon = mon, day
	}
	n, ok := dayNumber(day)
	if !ok || months[mon] == 0 {
		return repeatMust
	}
	if !exists(2028, int(months[mon]), n) {
		return "must be on a real day: " + strings.Join(rest, " ") + " is not one"
	}
	r.month, r.monthDay = months[mon], n
	return ""
}

func dayNumber(s string) (int, bool) {
	if m := ordinalRe.FindStringSubmatch(s); m != nil {
		s = m[1]
	}
	if !isNumber(s) {
		return 0, false
	}
	return atoi(s), true
}
