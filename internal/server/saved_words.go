package server

import (
	"strings"

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
		on, ok := clean[name].(bool)
		if !ok {
			return o
		}
		title := strings.TrimSpace(titleOf(t, rec))
		if title == "" {
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
