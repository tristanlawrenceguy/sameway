package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A tag the action gives is checked, when it asks, against what the person
// did with the latest records given the same tag: kept it, took it off,
// changed it to another, or nothing yet. Only the person's own choices are
// shown, never a tag the assistant kept for them, so what it follows is
// their judgement, not an echo of its own.

// verdict is what the check decided about one tag.
type verdict struct {
	Decide string `json:"decide"` // keep, confirm, drop or change
	To     string `json:"to"`
	Why    string `json:"why"`
}

const judgeSystem = `You check a tag you suggested for a record against what the person did with the latest records given the same tag. Follow their choices, not your own view:
- "confirm" only when they kept this tag on records plainly like this one, every time;
- "drop" when they took it off records plainly like this one;
- "change" when they changed it to the same other tag on records plainly like this one, with "to" that tag;
- otherwise "keep", leaving it for the person to decide.
Answer with JSON only: {"decide": "keep" | "confirm" | "drop" | "change", "to": "...", "why": "the past choice you followed, in a few words"}`

// judge is the check for one tag; with no past choices, or a model that
// cannot say, it is keep.
func (s *Service) judge(ctx context.Context, t *schema.Type, rec *store.Record, words string, p chosen, defs []tagDef, most int) verdict {
	past := s.pastChoices(p.Tag, t.Name+"/"+rec.ID, most)
	if len(past) == 0 {
		return verdict{Decide: "keep"}
	}
	var others []string
	for _, d := range defs {
		if !strings.EqualFold(d.name, p.Tag) {
			others = append(others, d.name)
		}
	}
	q := fmt.Sprintf("You suggested the tag %q (%s) for this record:\n%s\n\nThe other tags are: %s.\n\nWhat the person did with the latest records given %q, newest first:\n%s",
		p.Tag, p.Why, clipRunes(words, 1500), strings.Join(others, ", "), p.Tag, strings.Join(past, "\n"))
	var v verdict
	if err := s.askJSON(ctx, judgeSystem, q, &v); err != nil {
		return verdict{Decide: "keep"}
	}
	v.Decide = strings.ToLower(strings.TrimSpace(v.Decide))
	if v.Decide == "change" {
		want := strings.TrimSpace(v.To)
		v.To = ""
		for _, d := range defs {
			if strings.EqualFold(d.name, want) {
				v.To = d.name
			}
		}
	}
	switch {
	case v.Decide == "change" && v.To == "":
		v.Decide = "keep" // to a tag that is not one: the person decides
	case v.Decide != "confirm" && v.Decide != "drop" && v.Decide != "change":
		v.Decide = "keep"
	}
	return v
}

// pastChoices are the latest records given a tag, newest first, each with
// what the person did, at most most; the record being tagged is not one.
func (s *Service) pastChoices(tag, not string, most int) []string {
	cls, err := s.Store.List(ClassificationType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 200})
	if err != nil {
		return nil
	}
	var out []string
	for _, c := range cls {
		if len(out) == most {
			break
		}
		if !strings.EqualFold(fmt.Sprint(c.Fields["tag"]), tag) || c.Fields["record"] == not || c.Fields["confirmed_by"] == "judgement" {
			continue
		}
		typ, id, _ := strings.Cut(fmt.Sprint(c.Fields["record"]), "/")
		t, ok := s.Store.Types().Get(typ)
		rec, err := s.Store.Get(typ, id)
		if !ok || err != nil {
			continue
		}
		did := map[string]string{"confirmed": "kept it", "removed": "took the tag off", "changed": "changed it to " + fmt.Sprint(c.Fields["to"]), "suggested": "has done nothing with it yet"}[fmt.Sprint(c.Fields["state"])]
		out = append(out, fmt.Sprintf("- %q: the person %s", clipRunes(strings.ReplaceAll(recordWords(s.Store, t, rec), "\n", " "), 240), did))
	}
	return out
}

// tagsChanged keeps what the person did with a suggested tag: taken off
// the record, it was removed; taken off with another of their tags put on
// in its place, it was changed to that one.
func (s *Service) tagsChanged(t *schema.Type, was, now *store.Record) {
	if t.Internal || was == nil || now == nil {
		return
	}
	if _, ok := t.Field("tags"); !ok {
		return
	}
	before, after := map[string]bool{}, map[string]bool{}
	for _, v := range records.StringList(was.Fields["tags"]) {
		before[strings.ToLower(v)] = true
	}
	for _, v := range records.StringList(now.Fields["tags"]) {
		after[strings.ToLower(v)] = true
	}
	var added []string
	for _, d := range s.tagDefs(nil) {
		if after[strings.ToLower(d.name)] && !before[strings.ToLower(d.name)] {
			added = append(added, d.name)
		}
	}
	cls, err := s.Store.List(ClassificationType, store.ListOptions{})
	if err != nil {
		return
	}
	for _, c := range cls {
		tag := strings.ToLower(fmt.Sprint(c.Fields["tag"]))
		if c.Fields["record"] != t.Name+"/"+now.ID || c.Fields["state"] != "suggested" || after[tag] || !before[tag] {
			continue
		}
		change := map[string]any{"state": "removed"}
		if len(added) == 1 {
			change = map[string]any{"state": "changed", "to": added[0]}
		}
		records.ApplyOps(s.Store, records.Op{Type: ClassificationType, ID: c.ID, After: change})
	}
}
