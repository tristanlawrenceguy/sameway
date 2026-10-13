package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// keepOneOp makes records that are one thing twice into one: the one the
// person means is kept and the others, open and called the same
// (records.Alike), are merged into it (records.KeepOne). A list offers the
// same for those nothing tells apart; this is for when the assistant is
// told, or sees, that the rest are the same thing.
var keepOneOp = Op{Title: "Keep one of records alike", Traits: Traits{Destructive: true},
	Words: []string{"duplicate", "duplicates", "twice", "same", "merge", "tidy"},
	Doing: func(a callArgs) string { return "Merging" + or(" "+schema.Plural(a.Type), " records") },
	Run:   func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.keepOne(a.Type, a.ID) },
	Tool: llm.Tool{Name: "keep_one",
		Description: "Make records that are one thing twice into one: keep the one given, and merge into it every other open record of its type with the same name. What it leaves empty is filled from them, whatever pointed at them points at it, and they go. One change, undone in one go. Only when they are the same thing, not when they are the same errand on different days.",
		Schema: obj(map[string]any{
			"type": typeArg,
			"id":   map[string]any{"type": "string", "description": "The one to keep."},
		}, "type", "id")},
	Offered: typed}

func (s *Service) keepOne(typeName, id string) toolResult {
	t, err := records.ContentType(s.Store, typeName)
	if err != nil {
		return fail("%v", err)
	}
	kept, err := s.Store.Get(t.Name, id)
	if err != nil {
		return fail("no %s with id %s. Use find_records to get the id", t.Name, id)
	}
	others := records.Alike(s.Store, t, kept)
	if len(others) == 0 {
		return fail("no other open %s is called as %s is: there is nothing to merge", schema.Words(t.Name), id)
	}
	done, err := (&records.Book{Store: s.Store}).Apply(records.KeepOne(s.Store, t, kept, others)...)
	if err != nil {
		return fail("nothing merged: %v", err)
	}
	title := records.Name(s.Store, t, kept)
	c := records.Change{Action: "merged", Component: t.Name, ID: kept.ID, Href: "/t/" + t.Name + "/" + kept.ID,
		Detail: fmt.Sprintf("%s, %d into one", title, len(others)+1), Ops: done}
	return toolResult{text: fmt.Sprintf("merged %d %s called %q into %s, at %s.", len(others)+1, schema.Plural(t.Name), title, kept.ID, c.Href), change: &c}
}
