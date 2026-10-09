package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Suggesting is an action like any other: set off by a record (an email
// tagged to do, a booking that came in, a note), it fills a record of the
// kind it names from it (a task, an event, a reminder, a person, any kind
// the workspace has) and puts it to the person as a question, made with a
// press, changed, or turned down. Nothing is made without them. With check,
// it is shown what the person did with its last suggestions: made as they
// were, made and then changed, or turned down, to follow them.

// suggestion is what the model makes of a record, for one kind.
type suggestion struct {
	Make   bool           `json:"make"`
	Fields map[string]any `json:"fields"`
	Why    string         `json:"why"`
}

const suggestSystem = `You read one record that came to a person (an email, a message) or that they wrote, and suggest one %s for them from it, only when the record asks for or describes one; otherwise say make false.
Fill only these fields, and leave out any the record does not say:
%s
Answer with JSON only: {"make": true or false, "fields": {...}, "why": "four to eight words"}`

// fieldGuide says how the model writes each field of the kind it makes:
// a day in the record's own words, for Sameway to count; a person by name.
func (s *Service) fieldGuide(t *schema.Type) string {
	var b strings.Builder
	for _, f := range t.Shown() {
		if f.ReadOnly || ServerFilled(f.Description) || f.RefList() || f.Type == "json" || f.Name == "tags" {
			continue
		}
		how := fieldKind(f)
		switch {
		case f.Type == "datetime":
			how = "the words of the record that say the day, copied exactly, as short as they can be (\"Friday\", \"15 October at 14:30\", \"the 1st\"); do not work out a date"
		case f.Type == "ref" && f.To == records.PersonType:
			how = "the name of a person named in the record"
		case f.Type == "ref":
			continue
		case f.Name == t.Title:
			how = "a short title starting with a verb for a thing to do, else a short name, in the record's language"
		}
		desc := ""
		if f.Description != "" {
			desc = ": " + trim.Line(f.Description, 100)
		}
		fmt.Fprintf(&b, "- %s (%s)%s\n", f.Name, how, desc)
	}
	return b.String()
}

// Suggest puts to the person a record of the kind the action makes,
// filled from the record that set it off, and says what it put.
func (s *Service) Suggest(ctx context.Context, action *store.Record, typeName, id string) toolResult {
	from, ok := s.Store.Types().Get(typeName)
	rec, err := s.Store.Get(typeName, id)
	if !ok || err != nil {
		return fail("a suggest action reads the record that sets it off; set when and what so a record does")
	}
	makeName, _ := action.Fields["make"].(string)
	mt, ok := s.Store.Types().Get(makeName)
	if !ok || mt.Internal {
		var kinds []string
		for _, t := range records.ContentTypes(s.Store) {
			kinds = append(kinds, t.Name)
		}
		return fail("a suggest action says what it makes: one of %s", strings.Join(kinds, ", "))
	}
	words := recordWords(s.Store, from, rec)
	q := "The record:\n" + words
	if check, _ := action.Fields["check"].(bool); check {
		if past := s.pastSuggestions(action.ID, examplesOf(action)); len(past) > 0 {
			q += "\n\nWhat the person did with your last suggestions from this action, newest first. Follow their choices: what they turned down, do not suggest again for records like it; what they changed, write as they changed it.\n" + strings.Join(past, "\n")
		}
	}
	var sug suggestion
	if err := s.askJSON(ctx, fmt.Sprintf(suggestSystem, schema.Words(mt.Name), s.fieldGuide(mt)), q, &sug); err != nil {
		return fail("could not read %s: %v", records.Name(s.Store, from, rec), err)
	}
	name := records.Name(s.Store, from, rec)
	if !sug.Make {
		return toolResult{text: name + ": nothing to suggest"}
	}
	fields := s.fillSuggested(mt, sug.Fields, sentOf(rec))
	if t, _ := fields[mt.Title].(string); strings.TrimSpace(t) == "" {
		return toolResult{text: name + ": nothing to suggest"}
	}
	summary := fmt.Sprintf("Make a %s from %q: %s?", schema.Words(mt.Name), name, s.suggestedWords(mt, fields))
	p, err := s.Store.Create(records.ProposalType, map[string]any{"summary": summary, "detail": strings.TrimSpace(sug.Why),
		"action": map[string]any{"tool": "create_record", "type": mt.Name, "fields": fields}, "state": "pending",
		"from": from.Name + "/" + rec.ID, "by_action": action.ID, "yes": "Make it", "no": "No"})
	if err != nil {
		return fail("could not put the suggestion: %v", err)
	}
	return toolResult{text: "suggested: " + summary, answer: p.ID, change: &records.Change{Action: "proposed", ID: p.ID, Detail: trim.Line(summary, 80)}}
}

// fillSuggested turns what the model wrote into a record's fields: a day
// counted from when the record was sent, a person found by name, and only
// fields the kind has, with values it takes.
func (s *Service) fillSuggested(t *schema.Type, given map[string]any, sent time.Time) map[string]any {
	out := map[string]any{}
	for name, v := range given {
		f, ok := t.Field(name)
		if !ok || f.ReadOnly {
			continue
		}
		str, isStr := v.(string)
		switch {
		case f.Type == "datetime" && isStr:
			if at, day, ok := when.Parse(dayWords(str), sent); ok && (at.Format("2006-01-02") != sent.Format("2006-01-02") || saysToday(str)) {
				if at.Before(sent.AddDate(0, 0, -1)) && !yearSaid.MatchString(str) {
					at = at.AddDate(1, 0, 0) // "by 31 January", sent in October: the coming one
				}
				out[name] = when.Store(at, day)
			}
		case f.Type == "ref" && f.To == records.PersonType && isStr:
			if p := s.PersonByName(strings.TrimSpace(str)); p != nil {
				out[name] = p.ID
			}
		case f.Type == "enum" && isStr:
			for _, val := range f.Values {
				if strings.EqualFold(val, strings.TrimSpace(str)) {
					out[name] = val
				}
			}
		case f.Type == "ref", f.RefList(), f.Type == "json":
		default:
			out[name] = v
		}
	}
	return out
}

// suggestedWords is a suggested record in a line: its title, and a day.
func (s *Service) suggestedWords(t *schema.Type, fields map[string]any) string {
	parts := []string{fmt.Sprint(fields[t.Title])}
	for _, f := range t.Fields {
		if v, ok := fields[f.Name].(string); ok && f.Type == "datetime" {
			parts = append(parts, schema.Words(f.Name)+" "+when.Relative(v, s.Now(), s.Setting != nil && s.Setting("ui.clock") == "24"))
		}
	}
	return strings.Join(parts, ", ")
}

// sentOf is when a record came: an email's sending, else its making.
func sentOf(rec *store.Record) time.Time {
	if v, _ := rec.Fields["received"].(string); v != "" {
		if at, _, ok := when.Parse(v, rec.CreatedAt); ok {
			return at.Local()
		}
	}
	return rec.CreatedAt.Local()
}

func examplesOf(action *store.Record) int {
	switch n := action.Fields["examples"].(type) {
	case float64:
		return max(int(n), 1)
	case int64:
		return max(int(n), 1)
	case int:
		return max(n, 1)
	}
	return 10
}

// pastSuggestions are the latest suggestions an action made and what the
// person did: made as it was, made then changed (and how), or turned down.
func (s *Service) pastSuggestions(action string, most int) []string {
	ps, err := s.Store.List(records.ProposalType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 200})
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range ps {
		if len(out) == most {
			break
		}
		if p.Fields["by_action"] != action || p.Fields["state"] == "pending" {
			continue
		}
		act, _ := p.Fields["action"].(map[string]any)
		said, _ := json.Marshal(act["fields"])
		source := fmt.Sprint(p.Fields["from"])
		if typ, id, ok := strings.Cut(source, "/"); ok {
			if t, ok := s.Store.Types().Get(typ); ok {
				if r, err := s.Store.Get(typ, id); err == nil {
					source = clipRunes(strings.ReplaceAll(recordWords(s.Store, t, r), "\n", " "), 200)
				}
			}
		}
		did := "turned it down"
		if p.Fields["state"] == "accepted" {
			did = "made it as it was" + s.changedSince(p, act)
		}
		out = append(out, fmt.Sprintf("- from %q you suggested %s; the person %s", source, said, did))
	}
	return out
}

// changedSince is how the person changed what was made from a suggestion,
// in words, or "".
func (s *Service) changedSince(p *store.Record, act map[string]any) string {
	typ, id, ok := strings.Cut(fmt.Sprint(p.Fields["made"]), "/")
	if !ok {
		return ""
	}
	made, err := s.Store.Get(typ, id)
	if err != nil {
		return ", then deleted it"
	}
	given, _ := act["fields"].(map[string]any)
	var diffs []string
	for k, was := range given {
		if now := made.Fields[k]; fmt.Sprint(now) != fmt.Sprint(was) {
			diffs = append(diffs, fmt.Sprintf("%s from %v to %v", k, was, now))
		}
	}
	if len(diffs) == 0 {
		return ""
	}
	return ", then changed " + strings.Join(diffs, ", ")
}

var yearSaid = regexp.MustCompile(`(19|20)\d\d`)

// saysToday is whether words name the day they were written on: "now" and
// "when you can" are no day, and a small model gave them as one.
func saysToday(s string) bool {
	s = strings.ToLower(s)
	return strings.Contains(s, "today") || strings.Contains(s, "tonight") || strings.Contains(s, "this morning") || strings.Contains(s, "this afternoon") || strings.Contains(s, "this evening")
}

// dayWords is a day as a record says it, without what says how it stands
// to the thing: "by Friday", "end of day Thursday", "Saturday morning".
func dayWords(s string) string {
	s = strings.ToLower(strings.Trim(strings.TrimSpace(s), ".,!"))
	for _, lead := range []string{"by ", "on ", "before ", "until ", "till ", "due ", "end of day ", "the end of ", "end of ", "from ", "no later than "} {
		s = strings.TrimPrefix(s, lead)
	}
	for _, tail := range []string{" morning", " afternoon", " evening", " night", " at the latest", " latest", " eod"} {
		s = strings.TrimSuffix(s, tail)
	}
	return strings.TrimSpace(s)
}
