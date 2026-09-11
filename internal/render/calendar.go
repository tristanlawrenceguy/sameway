package render

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Date maths for the calendar component. A month grid cannot be built in a
// template and must not need JavaScript, so the shape of the month is
// computed here and the template only lays it out.

// CalDay is one cell of a month grid.
type CalDay struct {
	// Date is the ISO day, empty for the padding cells around the month.
	Date string
	// Day is the day of the month, 0 for padding.
	Day int
	// Weekday is the full name, for the cell's accessible text.
	Weekday string
}

// monthWeeks returns the weeks of a month as rows of seven days, padded at
// each end so every row is a full week. month is "2026-09"; start is
// "monday" or "sunday".
func monthWeeks(month, start string) [][]CalDay {
	first, err := time.Parse("2006-01", strings.TrimSpace(month))
	if err != nil {
		return nil
	}
	offset := weekStart(start)
	// How many padding cells before the first of the month.
	lead := (int(first.Weekday()) - offset + 7) % 7
	last := first.AddDate(0, 1, -1).Day()

	var weeks [][]CalDay
	week := make([]CalDay, 0, 7)
	for i := 0; i < lead; i++ {
		week = append(week, CalDay{})
	}
	for day := 1; day <= last; day++ {
		d := first.AddDate(0, 0, day-1)
		week = append(week, CalDay{Date: d.Format("2006-01-02"), Day: day, Weekday: d.Weekday().String()})
		if len(week) == 7 {
			weeks = append(weeks, week)
			week = make([]CalDay, 0, 7)
		}
	}
	for len(week) > 0 && len(week) < 7 {
		week = append(week, CalDay{})
	}
	if len(week) == 7 {
		weeks = append(weeks, week)
	}
	return weeks
}

func weekStart(start string) int {
	if strings.EqualFold(strings.TrimSpace(start), "sunday") {
		return int(time.Sunday)
	}
	return int(time.Monday)
}

// monthName is the heading for a month: "September 2026".
func monthName(month string) string {
	t, err := time.Parse("2006-01", strings.TrimSpace(month))
	if err != nil {
		return month
	}
	return t.Format("January 2006")
}

// weekdayNames lists the seven column headings in order, each as the short
// form and the full name, so a table header can show one and say the other.
func weekdayNames(start string) []Weekday {
	offset := weekStart(start)
	out := make([]Weekday, 0, 7)
	base := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC) // a Sunday
	for i := 0; i < 7; i++ {
		d := base.AddDate(0, 0, offset+i)
		out = append(out, Weekday{Short: d.Format("Mon"), Full: d.Weekday().String()})
	}
	return out
}

// Weekday is a column heading: what is shown, and what is announced.
type Weekday struct{ Short, Full string }

// eventsOn picks the events falling on one ISO day. Events are the plain
// maps a component's props carry, each with a date and a label.
func eventsOn(events any, date string) []map[string]any {
	list, ok := events.([]any)
	if !ok || date == "" {
		return nil
	}
	var out []map[string]any
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		if d, _ := m["date"].(string); d == date {
			out = append(out, m)
		}
	}
	return out
}

// longDate is the spoken form of an ISO day: "Friday 11 September 2026".
func longDate(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return fmt.Sprintf("%s %d %s", t.Weekday(), t.Day(), t.Format("January 2006"))
}

// upcoming returns the next n events on or after a day, in date order. It
// is what the smallest sizes of a calendar show: at a glance, how many are
// coming; in brief, which ones.
func upcoming(events any, from string, count any) []map[string]any {
	// The normaliser hands integers back as int64, so take the count as it
	// comes rather than making every caller convert.
	n := 0
	switch v := count.(type) {
	case int:
		n = v
	case int64:
		n = int(v)
	case float64:
		n = int(v)
	}
	list, ok := events.([]any)
	if !ok {
		return nil
	}
	var out []map[string]any
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		d, _ := m["date"].(string)
		if d == "" || (from != "" && d < from) {
			continue
		}
		out = append(out, m)
	}
	sort.SliceStable(out, func(i, j int) bool {
		di, _ := out[i]["date"].(string)
		dj, _ := out[j]["date"].(string)
		if di != dj {
			return di < dj
		}
		ti, _ := out[i]["time"].(string)
		tj, _ := out[j]["time"].(string)
		return ti < tj
	})
	if n > 0 && len(out) > n {
		out = out[:n]
	}
	return out
}

// shortDate is the compact spoken form of a day: "Fri 11 Sep".
func shortDate(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.Format("Mon 2 Jan")
}
