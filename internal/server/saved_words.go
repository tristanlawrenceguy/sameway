package server

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/when"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// savedWords is what a saved edit says. A mark ticked says what it made
// true, "Order compost is done.", so ticking down a list is heard as each
// thing, not "Changes saved" again and again; its Undo names the thing.
// Any other edit is "Changes saved".
func savedWords(t *schema.Type, rec *store.Record, fields, clean map[string]any, undo string) outcome {
	o := outcome{Title: "Changes saved", Undo: undo}
	if len(fields) != 1 {
		return o
	}
	for name := range fields {
		// A day or a moment says how its words were read: "Due is Fri 2
		// Oct 2026, 14:00." The words were the person's; this is Sameway's.
		if f, ok := t.Field(name); ok && f.Type == "datetime" {
			if v, _ := clean[name].(string); v != "" {
				o.Text = fieldLabel(*f) + " is " + when.Text(v) + "."
			}
			return o
		}
		// How often, said back as it was read: "Repeat is every Tuesday."
		if f, ok := t.Field(name); ok && f.Type == "repeat" {
			o.Text = fieldLabel(*f) + " is " + repeatSaid(clean[name]) + "."
			return o
		}
		// A choice changed, as a board's Move does, says from where to
		// where: "Order compost moved from To do to Done."
		if f, ok := t.Field(name); ok && f.Type == "enum" {
			// Moved to Done, a task that repeats is due again at once.
			if title := strings.TrimSpace(titleOf(t, rec)); title != "" && t.Advanced(fields, clean) {
				o.Title, o.Text, o.Of = title+" is done.", dueAgain(t, clean), title
				return o
			}
			was, _ := rec.Fields[name].(string)
			now, _ := clean[name].(string)
			if title := strings.TrimSpace(titleOf(t, rec)); title != "" && now != "" && was != now {
				o.Title = title + " moved to " + f.ValueLabel(now) + "."
				if was != "" {
					o.Title = title + " moved from " + f.ValueLabel(was) + " to " + f.ValueLabel(now) + "."
				}
				o.Of = title
			}
			return o
		}
		on, ok := clean[name].(bool)
		if !ok {
			return o
		}
		title := strings.TrimSpace(titleOf(t, rec))
		if title == "" {
			return o
		}
		// Ticked, a task that repeats is done and due again at once:
		// "Water the ferns is done. It repeats every Tuesday, so it is due
		// again Tue 6 Oct 2026."
		if t.Advanced(fields, clean) {
			o.Title, o.Text, o.Of = title+" is done.", dueAgain(t, clean), title
			return o
		}
		word := strings.ToLower(label(name))
		if name == "show" {
			word = "shown"
		}
		if on {
			o.Title = title + " is " + word + "."
		} else {
			o.Title = title + " is not " + word + "."
		}
		o.Of = title
	}
	return o
}

// repeatSaid is a stored repeat in words, or once for none.
func repeatSaid(v any) string {
	if rule, _ := v.(string); rule != "" {
		return when.RepeatText(rule)
	}
	return "once, not again"
}

// dueAgain says when a record that repeats is due again, as a sentence;
// nothing for one that does not repeat.
func dueAgain(t *schema.Type, fields map[string]any) string {
	repeat, day, ok := t.Repeats()
	rule, _ := fields[repeat].(string)
	next, _ := fields[day].(string)
	if !ok || rule == "" || next == "" {
		return ""
	}
	return "It repeats " + when.RepeatText(rule) + ", so it is due again " + when.Text(next) + "."
}
