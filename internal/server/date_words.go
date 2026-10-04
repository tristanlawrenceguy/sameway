package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// dayWords is the record's first day as a few words after its title, the
// field named: due tomorrow, with the year when it is not this one.
func dayWords(t *schema.Type, rec *store.Record) string {
	for _, f := range t.Shown() {
		if f.Type != "datetime" {
			continue
		}
		v := str(rec.Fields[f.Name], f.Name)
		if v == "" || v == f.Name {
			continue
		}
		return "due " + whenWords(v)
	}
	for _, f := range t.Shown() {
		if f.Type != "time" {
			continue
		}
		v := str(rec.Fields[f.Name], f.Name)
		if v == "" || v == f.Name {
			continue
		}
		return "due " + whenWords(v)
	}
	for _, f := range t.Shown() {
		if f.Type != "date" {
			continue
		}
		v := str(rec.Fields[f.Name], f.Name)
		if v == "" || v == f.Name {
			continue
		}
		return "on " + whenWords(v)
	}
	return ""
}

// whenWords is a stored day or moment in natural language: today,
// yesterday, tomorrow, or the date with time for older entries.
func whenWords(v string) string {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return ""
	}
	now := time.Now()
	if strings.HasSuffix(v, "T00:00:00Z") {
		return shortDay2(ts.UTC(), now)
	}
	return relativeMoment(ts, now)
}

func relativeMoment(at time.Time, now time.Time) string {
	nowUTC := now.UTC()
	local := at.UTC()
	today := time.Date(nowUTC.Year(), nowUTC.Month(), nowUTC.Day(), 0, 0, 0, 0, time.UTC)
	d := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
	diff := int(d.Sub(today).Hours() / 24)

	clock := formatClockLocal(at)

	switch {
	case diff == 0:
		return "Today at " + clock
	case diff < 0 && diff >= -6:
		if diff == -1 {
			return "Yesterday at " + clock
		}
		return fmt.Sprintf("%d days ago at %s", -diff, clock)
	case diff > 0 && diff <= 6:
		if diff == 1 {
			return "Tomorrow at " + clock
		}
		return fmt.Sprintf("In %d days at %s", diff, clock)
	default:
		if local.Year() == nowUTC.Year() {
			return d.Format("2 Jan") + " at " + clock
		}
		return fmt.Sprintf("%s at %s %d", d.Format("2 Jan"), clock, local.Year())
	}
}

// momentWords is a stored moment in natural language: today at 3pm, yesterday at 5am.
func momentWords(at time.Time) string {
	return relativeMoment(at.UTC(), time.Now())
}

// shortDay formats a day-only value in natural language: Today, Yesterday, Tomorrow, or the date for older days.
func shortDay(d time.Time) string {
	return shortDay2(d, time.Now())
}

func shortDay2(d time.Time, now time.Time) string {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	nd := time.Date(d.UTC().Year(), d.UTC().Month(), d.UTC().Day(), 0, 0, 0, 0, now.Location())
	diff := int(nd.Sub(today).Hours() / 24)

	switch {
	case diff == 0:
		return "Today"
	case diff < 0 && diff >= -6:
		if diff == -1 {
			return "Yesterday"
		}
		return nd.Format("Mon 2 Jan")
	case diff > 0 && diff <= 6:
		if diff == 1 {
			return "Tomorrow"
		}
		return fmt.Sprintf("In %d days", diff)
	default:
		if nd.Year() == now.Year() {
			return nd.Format("2 Jan")
		}
		return nd.Format("2 Jan 2006")
	}
}

// formatClockLocal formats a time as 12-hour "H:MMam/pm" in the given zone.
func formatClockLocal(t time.Time) string {
	h := t.Hour()
	min := t.Minute()
	suffix := "am"
	if h >= 12 {
		suffix = "pm"
	}
	h12 := h % 12
	if h12 == 0 {
		h12 = 12
	}
	return fmt.Sprintf("%d:%02d%s", h12, min, suffix)
}
