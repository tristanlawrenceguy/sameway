package convert

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // a calendar's time zones, on any computer
)

// A calendar file (.ics, iCalendar, RFC 5545), as Google Calendar,
// Outlook and Apple Calendar all export it: its events, each with when it
// starts and ends, where, how often it happens again (an RRULE, kept as
// Sameway keeps repeats), and the calendar's own id for it. A cancelled
// event is left out, and so is a changed single time of a repeating one,
// which the repeating one already stands for.

// Event is one event in a calendar.
type Event struct {
	UID, Title, Where, Notes, Repeat string
	Starts, Ends                     string // RFC 3339, or a date alone for all day
}

// ParseICS reads the events of a calendar file.
func ParseICS(data []byte, local *time.Location) []Event {
	if local == nil {
		local = time.Local
	}
	// Long lines are folded: a line starting with a space or a tab goes on
	// the one before.
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n ", "")
	text = strings.ReplaceAll(text, "\n\t", "")
	var out []Event
	var ev *Event
	skip := false
	for _, line := range strings.Split(text, "\n") {
		name, params, value := property(line)
		switch {
		case name == "BEGIN" && strings.EqualFold(value, "VEVENT"):
			ev, skip = &Event{}, false
		case name == "END" && strings.EqualFold(value, "VEVENT"):
			if ev != nil && !skip && (ev.Title != "" || ev.Starts != "") {
				if ev.Title == "" {
					ev.Title = "Event"
				}
				out = append(out, *ev)
			}
			ev = nil
		case ev == nil:
		case name == "SUMMARY":
			ev.Title = unescape(value)
		case name == "LOCATION":
			ev.Where = unescape(value)
		case name == "DESCRIPTION":
			ev.Notes = unescape(value)
		case name == "UID":
			ev.UID = value
		case name == "RRULE":
			ev.Repeat = value
		case name == "DTSTART":
			ev.Starts = icsTime(value, params, local)
		case name == "DTEND":
			ev.Ends = icsTime(value, params, local)
		case name == "STATUS" && strings.EqualFold(value, "CANCELLED"), name == "RECURRENCE-ID":
			skip = true
		}
	}
	return out
}

// property splits NAME;PARAM=x;PARAM=y:value.
func property(line string) (name string, params map[string]string, value string) {
	line = strings.TrimRight(line, "\r")
	head, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", nil, ""
	}
	parts := strings.Split(head, ";")
	params = map[string]string{}
	for _, p := range parts[1:] {
		if k, v, ok := strings.Cut(p, "="); ok {
			params[strings.ToUpper(k)] = strings.Trim(v, `"`)
		}
	}
	return strings.ToUpper(parts[0]), params, value
}

// icsTime reads 20261003 (a day), 20261003T140000Z (UTC), or a local time
// in its TZID, else in the computer's own zone.
func icsTime(v string, params map[string]string, local *time.Location) string {
	v = strings.TrimSpace(v)
	if params["VALUE"] == "DATE" || len(v) == 8 {
		if t, err := time.Parse("20060102", v); err == nil {
			return t.Format("2006-01-02")
		}
		return ""
	}
	if strings.HasSuffix(v, "Z") {
		if t, err := time.Parse("20060102T150405Z", v); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
		return ""
	}
	loc := local
	if tz := params["TZID"]; tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		} else if l, ok := windowsZones[tz]; ok {
			if l, err := time.LoadLocation(l); err == nil {
				loc = l
			}
		}
	}
	if t, err := time.ParseInLocation("20060102T150405", v, loc); err == nil {
		return t.Format(time.RFC3339)
	}
	return ""
}

// windowsZones are the zone names Outlook writes, for the most common.
var windowsZones = map[string]string{
	"GMT Standard Time": "Europe/London", "W. Europe Standard Time": "Europe/Berlin",
	"Romance Standard Time": "Europe/Paris", "Central Europe Standard Time": "Europe/Budapest",
	"E. Europe Standard Time": "Europe/Chisinau", "Eastern Standard Time": "America/New_York",
	"Central Standard Time": "America/Chicago", "Mountain Standard Time": "America/Denver",
	"Pacific Standard Time": "America/Los_Angeles", "AUS Eastern Standard Time": "Australia/Sydney",
	"India Standard Time": "Asia/Kolkata", "China Standard Time": "Asia/Shanghai",
	"Tokyo Standard Time": "Asia/Tokyo", "UTC": "UTC",
}

func unescape(s string) string {
	r := strings.NewReplacer(`\n`, "\n", `\N`, "\n", `\,`, ",", `\;`, ";", `\\`, `\`)
	return strings.TrimSpace(r.Replace(s))
}

// icsMarkdown is a calendar file as a person reads it: each event, when,
// and where.
func icsMarkdown(data []byte) (string, error) {
	events := ParseICS(data, nil)
	if len(events) == 0 {
		return "", fmt.Errorf("the calendar has no events in it")
	}
	var b strings.Builder
	for _, e := range events {
		fmt.Fprintf(&b, "- **%s**", e.Title)
		if e.Starts != "" {
			b.WriteString(", " + icsWhen(e.Starts))
		}
		if e.Where != "" {
			b.WriteString(", at " + e.Where)
		}
		if e.Repeat != "" {
			b.WriteString(" (repeats)")
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

func icsWhen(v string) string {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.Local().Format("Mon 2 Jan 2006, 15:04")
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		return t.Format("Mon 2 Jan 2006")
	}
	return v
}
