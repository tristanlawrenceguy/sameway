package server

import (
	"html/template"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A record's day at a glance (design/foundations/glance.md): named by its
// field, said as a day a person plans by, late said in words, and kept in
// a <time> with its value and, when the words leave the date out, the date
// in full for a pointer.

// dayGlance is a record's first day as a person reads it. done is whether
// it is ticked.
func dayGlance(t *schema.Type, rec *store.Record, done bool, now time.Time) (glanceFact, bool) {
	for _, f := range t.Shown() {
		v, _ := rec.Fields[f.Name].(string)
		if f.Type != "datetime" || v == "" {
			continue
		}
		words := when.Relative(v, now)
		text, tone, class := words, "info", "sw-when"
		name := dayName(f)
		if name != "" {
			text = name + " " + afterLabel(words)
		}
		passed := dayPassed(v, now)
		switch {
		case passed && t.DoneField() != nil && !done:
			// Late is said in words, not only in amber (WCAG 1.4.1). A row
			// keeps its short day: its listing's Overdue heading says it.
			text, tone, class = "Overdue, "+afterLabel(words), "warning", class+" sw-when--past"
			if name != "" {
				text = "Overdue, " + lowerName(name) + " " + afterLabel(words)
			}
		case passed:
			tone = "neutral"
		case strings.HasPrefix(words, "Today") && !done:
			class += " sw-when--today"
		}
		full := ""
		if when.LeavesDateOut(words) {
			full = when.Full(v)
		}
		return glanceFact{Kind: "day", Field: f.Name, Text: text, Tone: tone, Short: words, Class: class, When: when.Machine(v), Full: full}, true
	}
	return glanceFact{}, false
}

// dayPassed is whether a stored day or moment is before now: a moment once
// its time has gone, a day once the reader's today is after it.
func dayPassed(v string, now time.Time) bool {
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return false
	}
	if when.IsDay(v) {
		return ts.UTC().Format("2006-01-02") < now.Format("2006-01-02")
	}
	return ts.Before(now)
}

// dayName is what a day is called before it, its field's label: Due,
// Starts, Goal by. A field called at or on is said by its words already
// (Today at 2pm), and is not named.
func dayName(f schema.Field) string {
	name := f.Display()
	switch strings.ToLower(name) {
	case "at", "on", "when", "date", "day", "time":
		return ""
	}
	return name
}

// lowerName lowers the first letter of what a day is called, after
// Overdue: Overdue, due Fri 2 Oct.
func lowerName(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// afterLabel lowers a phrase's first letter where it follows a label:
// Due tomorrow, not Due Tomorrow. A month or a day's name stays as it is.
func afterLabel(s string) string {
	for _, w := range []string{"Today", "Tomorrow", "Yesterday"} {
		if strings.HasPrefix(s, w) {
			return strings.ToLower(s[:1]) + s[1:]
		}
	}
	return s
}

// timeHTML puts what a day says in a <time> holding its value, with the
// date in full as its title when the words leave it out (Today). A screen
// reader reads the words, which say it on their own; the value is for a
// machine, the title for a pointer.
func timeHTML(class, value, full string, inner template.HTML) string {
	b := "<time"
	if class != "" {
		b += ` class="` + class + `"`
	}
	if value != "" {
		b += ` datetime="` + template.HTMLEscapeString(value) + `"`
	}
	if full != "" {
		b += ` title="` + template.HTMLEscapeString(full) + `"`
	}
	return b + ">" + string(inner) + "</time>"
}
