package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// keepOneOp makes records that are one thing twice into one: the one the
// person means is kept and the others are merged into it (records.KeepOne).
// Left to itself it merges what a list offers to, those nothing tells
// apart (records.TwinsOf), so a press and a call do the same; told which,
// it merges those of the ones called the same (records.Alike). The same
// errand on another day is never merged unless it is named.
var keepOneOp = Op{Title: "Keep one of records alike", Traits: Traits{Destructive: true},
	Doing: func(a callArgs) string { return "Merging" + or(" "+schema.Plural(a.Type), " records") },
	Run:   func(s *Service, a toolArgs, call llm.ToolCall) toolResult { return s.keepOne(a.Type, a.ID, a.Merge) },
	Tool: llm.Tool{Name: "keep_one",
		Description: "Make records that are one thing twice into one: keep the one given and merge into it others of its type with the same name. Without merge, those nothing tells apart from it (open, the same day or no day); merge names the ones that are the same thing, as the same errand on another day is two things. The kept one's empty fields are filled from them, their notes added, what pointed at them points at it, and they go. One change, one Undo.",
		Schema: obj(map[string]any{
			"type":  typeArg,
			"id":    map[string]any{"type": "string", "description": "The one to keep."},
			"merge": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Ids of the ones to merge in, open and called the same."},
		}, "type", "id")},
	Offered: typed}

func (s *Service) keepOne(typeName, id string, merge []string) toolResult {
	t, err := s.contentType(typeName)
	if err != nil {
		return fail("%v", err)
	}
	kept, err := s.Store.Get(t.Name, id)
	if err != nil {
		return fail("%s. Use find_records to get the id", records.NoRecord(t.Name, id))
	}
	alike := records.Alike(s.Store, t, kept)
	if len(alike) == 0 {
		return fail("no other open %s is called as %s is: there is nothing to merge", schema.Words(t.Name), id)
	}
	others, bad := chosen(alike, merge)
	if bad != "" {
		return fail("%s is not an open %s called as %s is; those are %s. Nothing merged", bad, schema.Words(t.Name), id, s.alikeSaid(t, alike))
	}
	if len(merge) == 0 {
		others = records.TwinsOf(s.Store, t, kept)
	}
	if len(others) == 0 {
		return fail("nothing is the same day as %s; called the same, on other days: %s. If one is the same thing, give it in merge. Nothing merged", id, s.alikeSaid(t, alike))
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

// chosen is the ones of alike named in ids, or the first id that is not
// one of them.
func chosen(alike []*store.Record, ids []string) ([]*store.Record, string) {
	byID := map[string]*store.Record{}
	for _, r := range alike {
		byID[r.ID] = r
	}
	var out []*store.Record
	for _, id := range ids {
		r, ok := byID[id]
		if !ok {
			return nil, id
		}
		delete(byID, id)
		out = append(out, r)
	}
	return out, ""
}
