package track

import (
	"strconv"
	"strings"
)

// Amount is a number as a person reads it: 3, 2.5, 12 km, 8 glasses.
func Amount(v float64, unit string) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if strings.Contains(s, ".") {
		s = strconv.FormatFloat(v, 'f', 1, 64)
		s = strings.TrimSuffix(s, ".0")
	}
	if unit == "" {
		return s
	}
	// One of a thing is said as one: 1 glass, 1 hour, not 1 glasses.
	if v == 1 {
		switch {
		case strings.HasSuffix(unit, "sses"):
			unit = strings.TrimSuffix(unit, "es")
		case strings.HasSuffix(unit, "s") && !strings.HasSuffix(unit, "ss") && len(unit) > 2:
			unit = strings.TrimSuffix(unit, "s")
		}
	}
	return s + " " + unit
}

// Progress says where a period stands: "3 of 8 glasses", "done",
// "13.5 of 21 hours, 7.5 left", "23 of 21 hours, 2 over", "72.4 kg".
func Progress(h Habit, s Summary) string {
	h = Normal(h)
	switch h.Aim {
	case Record:
		if !s.Logged {
			return "nothing yet"
		}
		return Amount(s.Now, h.Unit)
	case Limit:
		of := Amount(s.Now, "") + " of " + Amount(s.Target, h.Unit)
		if s.Over > 0 {
			return of + ", " + Amount(s.Over, "") + " over"
		}
		return of + ", " + Amount(s.Left, "") + " left"
	}
	if h.Unit == "" && s.Target == 1 {
		if s.Met {
			return "done"
		}
		return "not yet"
	}
	return Amount(s.Now, "") + " of " + Amount(s.Target, h.Unit)
}
