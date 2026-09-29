package server

import (
	"strings"
	"unicode/utf8"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// changedWords says what one edited field became and what it was, under
// "Changes saved": "Project changed from Garden to House.", "Target
// changed from 3 to 5.", "Notes changed." for text too long to repeat. A
// screen reader hears what changed, not only that something did, and the
// Undo beside it takes back that. Nothing is said when nothing changed.
func (s *Server) changedWords(f schema.Field, before, after any) string {
	was, now := s.shownAs(f, before), s.shownAs(f, after)
	if was == now {
		return ""
	}
	name := fieldLabel(f)
	switch {
	case !short(was) || !short(now) || f.Type == "markdown" || f.Type == "text" && (strings.Contains(was, "\n") || strings.Contains(now, "\n")):
		return name + " changed."
	case now == "":
		return name + " cleared; it was " + was + "."
	case was == "":
		return name + " set to " + now + "."
	}
	return name + " changed from " + was + " to " + now + "."
}

// shownAs is a value as a page shows it: a ref by the title it points at,
// a choice by its label.
func (s *Server) shownAs(f schema.Field, v any) string {
	text := strings.TrimSpace(display(f, v))
	switch f.Type {
	case "ref":
		return s.refTitle(f, text)
	case "enum":
		if text != "" {
			return f.ValueLabel(text)
		}
	}
	return text
}

// short is a value worth repeating in a sentence.
func short(v string) bool {
	return utf8.RuneCountInString(v) <= 60 && !strings.Contains(v, "\n")
}

// savedText adds what changed to a saved edit's words, when one field was
// edited and savedWords said no more than that it was saved.
func (s *Server) savedText(o outcome, t *schema.Type, rec *store.Record, fields, clean map[string]any) outcome {
	if len(fields) != 1 || o.Text != "" || o.Title != "Changes saved" {
		return o
	}
	for name := range fields {
		f, ok := t.Field(name)
		if !ok || f.Type == "bool" || f.Type == "datetime" || f.Type == "json" {
			return o
		}
		if text := s.changedWords(*f, rec.Fields[name], clean[name]); text != "" {
			o.Text = text
			if title := strings.TrimSpace(s.title(t, rec)); title != "" {
				o.Of = title
			}
		}
	}
	return o
}
