package track

import (
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/when"
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

// EntryName is what one entry is called wherever it is named, the same
// in a list, on its page and in the log: its habit and how much, "Read:
// 25 minutes", or "Stretch: done" for a plain done. With its habit gone,
// its note, or the day it was done. Never its id, which says nothing.
func EntryName(habit, unit string, fields map[string]any) string {
	if habit != "" {
		amount := 1.0
		switch n := fields["amount"].(type) {
		case float64:
			amount = n
		case int:
			amount = float64(n)
		case int64:
			amount = float64(n)
		}
		how := Amount(amount, unit)
		if unit == "" && amount == 1 {
			how = "done"
		}
		return habit + ": " + how
	}
	if note, _ := fields["note"].(string); strings.TrimSpace(note) != "" {
		return note
	}
	if at, _ := fields["at"].(string); at != "" {
		return "Entry, " + when.Text(at)
	}
	return "Entry"
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
