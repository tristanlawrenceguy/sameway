package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Classifying is an action like any other: set off by a record (an email
// coming in, a note added, a task changed), it gives that record the tags
// that fit, from the tags the person defined, each with why. A tag the
// action gives is a suggestion until the person keeps it, takes it off or
// changes it, and what they did is kept (ClassificationType), so the next
// time the action can be shown their choices and follow them: the
// assistant applying the person's judgement is still the person's
// judgement; its own is not, so without that it only suggests.

// ClassificationType is a tag an action gave a record, and what became of it.
const ClassificationType = "classification"

// TagType is a tag and what it means.
const TagType = "tag"

type tagDef struct{ name, means string }

// tagDefs are the tags an action may give: those it names, else all.
func (s *Service) tagDefs(only []string) []tagDef {
	recs, err := s.Store.List(TagType, store.ListOptions{OrderBy: "name"})
	if err != nil {
		return nil
	}
	want := map[string]bool{}
	for _, o := range only {
		want[strings.ToLower(strings.TrimSpace(o))] = true
	}
	var out []tagDef
	for _, r := range recs {
		name, _ := r.Fields["name"].(string)
		means, _ := r.Fields["means"].(string)
		name = strings.TrimSpace(name)
		if name == "" || len(want) > 0 && !want[strings.ToLower(name)] {
			continue
		}
		out = append(out, tagDef{name, strings.TrimSpace(means)})
	}
	return out
}

// recordWords is a record as the model reads it: its name, then each of
// its words, clipped.
func recordWords(st *store.Store, t *schema.Type, rec *store.Record) string {
	var b strings.Builder
	b.WriteString(records.Name(st, t, rec))
	for _, f := range t.Fields {
		if f.Name == "tags" || f.Name == t.Title {
			continue
		}
		if v, ok := rec.Fields[f.Name].(string); ok && strings.TrimSpace(v) != "" && (f.Type == "string" || f.Type == "text" || f.Type == "markdown" || f.Type == "datetime") {
			b.WriteString("\n" + schema.Words(f.Name) + ": " + v)
		}
	}
	return clipRunes(b.String(), 4000)
}

var anyJSON = regexp.MustCompile(`(?s)[\[{].*[\]}]`)

// askJSON puts one narrow question to the model, thinking off, and reads the
// JSON it answers with into out.
func (s *Service) askJSON(ctx context.Context, system, question string, out any) error {
	if s.Provider == nil {
		return fmt.Errorf("no model is connected")
	}
	resp, err := s.Provider.Complete(ctx, llm.Request{System: system, Messages: []llm.Message{{Role: llm.RoleUser, Content: question}}, MaxTokens: 2048, Effort: "none"})
	if err != nil {
		return err
	}
	raw := anyJSON.FindString(resp.Text)
	if raw == "" {
		return fmt.Errorf("the model answered without JSON: %.80s", resp.Text)
	}
	return json.Unmarshal([]byte(raw), out)
}

const classifySystem = `You tag a record for a person: an email or a message that came to them, or something they wrote. Each tag is defined by the person in their own words, where "I", "me" and "my" are the person, never you.
Read what the record asks of the person or is about, then give each tag whose definition is true of it, as written; none may be.
Answer with JSON only: {"tags": [{"tag": "...", "why": "four to eight words"}]}`

const oneTagSystem = `You decide whether one tag fits a record for a person: an email or a message that came to them, or something they wrote. The tag is defined by the person in their own words, where "I", "me" and "my" are the person, never you.
Read what the record asks of the person or is about, then say whether the definition is true of it.
Answer with JSON only: {"fits": true or false, "why": "four to eight words"}`

// chosen is a tag the model gave, and why.
type chosen struct {
	Tag string `json:"tag"`
	Why string `json:"why"`
}

// Classify gives the record the tags that fit, as the action says: from
// its tags, all at once or each apart, checked against the person's past
// choices when it asks. It says what it gave.
func (s *Service) Classify(ctx context.Context, action *store.Record, typeName, id string) toolResult {
	t, ok := s.Store.Types().Get(typeName)
	rec, err := s.Store.Get(typeName, id)
	if !ok || err != nil {
		return fail("a classify action tags the record that sets it off; set when and what so a record does")
	}
	defs := s.tagDefs(records.StringList(action.Fields["tags"]))
	if len(defs) == 0 {
		return fail("there are no tags to give: add some, each with what it means, at /t/%s", TagType)
	}
	words := recordWords(s.Store, t, rec)
	picks, err := s.pickTags(ctx, action, defs, words)
	if err != nil {
		return fail("could not tag %s: %v", records.Name(s.Store, t, rec), err)
	}
	check, _ := action.Fields["check"].(bool)
	most := 10
	switch n := action.Fields["examples"].(type) {
	case float64:
		most = int(n)
	case int64:
		most = int(n)
	case int:
		most = n
	}
	if most < 1 {
		most = 10
	}
	var said, given []string
	for _, p := range picks {
		state, confirmedBy, tag, why := "suggested", "", p.Tag, p.Why
		if check {
			v := s.judge(ctx, t, rec, words, p, defs, most) // judgement.go
			switch v.Decide {
			case "drop":
				said = append(said, tag+" (not given: "+v.Why+")")
				continue
			case "confirm":
				state, confirmedBy, why = "confirmed", "judgement", v.Why
			case "change":
				tag, why = v.To, v.Why
			}
		}
		s.giveTag(t, rec, tag)
		given = append(given, tag)
		cl := map[string]any{"record": t.Name + "/" + rec.ID, "tag": tag, "why": why, "action": action.ID, "state": state}
		if confirmedBy != "" {
			cl["confirmed_by"] = confirmedBy
		}
		records.ApplyOps(s.Store, records.Op{Type: ClassificationType, After: cl})
		said = append(said, tag+" ("+map[string]string{"suggested": "suggested", "confirmed": "kept, as you chose before"}[state]+": "+why+")")
	}
	title, _ := action.Fields["title"].(string)
	name := records.Name(s.Store, t, rec)
	if len(said) == 0 {
		return toolResult{text: name + ": no tag fits", answer: ""}
	}
	return toolResult{text: name + ": " + strings.Join(said, "; "), answer: strings.Join(given, ", "),
		change: &records.Change{Action: "tagged", Component: t.Name, ID: rec.ID, Detail: name + ": " + strings.Join(said, "; "), By: title, Href: "/t/" + t.Name + "/" + rec.ID}}
}

// giveTag puts a tag on a record that has a tags list, once.
func (s *Service) giveTag(t *schema.Type, rec *store.Record, tag string) {
	f, ok := t.Field("tags")
	if !ok || f.Type != "list" {
		return
	}
	if now, err := s.Store.Get(t.Name, rec.ID); err == nil {
		rec = now // as it is after the tags given just before
	}
	tags := records.StringList(rec.Fields["tags"])
	for _, have := range tags {
		if strings.EqualFold(have, tag) {
			return
		}
	}
	var out []any
	for _, have := range tags {
		out = append(out, have)
	}
	records.ApplyOps(s.Store, records.Op{Type: t.Name, ID: rec.ID, After: map[string]any{"tags": append(out, tag)}})
}

// pickTags asks which of the tags fit: all at once, or each on its own
// when the action says so.
func (s *Service) pickTags(ctx context.Context, action *store.Record, defs []tagDef, words string) ([]chosen, error) {
	var picks []chosen
	if apart, _ := action.Fields["apart"].(bool); apart {
		for _, d := range defs {
			var a struct {
				Fits bool   `json:"fits"`
				Why  string `json:"why"`
			}
			if err := s.askJSON(ctx, oneTagSystem, fmt.Sprintf("Tag %q means: %s\n\nThe record:\n%s", d.name, d.means, words), &a); err != nil {
				return nil, err
			}
			if a.Fits {
				picks = append(picks, chosen{d.name, a.Why})
			}
		}
	} else {
		var list strings.Builder
		for _, d := range defs {
			list.WriteString(fmt.Sprintf("- %s: %s\n", d.name, d.means))
		}
		var a struct {
			Tags []chosen `json:"tags"`
		}
		if err := s.askJSON(ctx, classifySystem, "The tags:\n"+list.String()+"\nThe record:\n"+words, &a); err != nil {
			return nil, err
		}
		known := map[string]string{}
		for _, d := range defs {
			known[strings.ToLower(d.name)] = d.name
		}
		for _, p := range a.Tags {
			if name, ok := known[strings.ToLower(strings.TrimSpace(p.Tag))]; ok {
				picks = append(picks, chosen{name, p.Why})
			}
		}
	}
	return picks, nil
}

func clipRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
