package chat

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/prose"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// Someone's own words are theirs to change. Asked for a change there is
// no reason to turn down (fix the spelling), the assistant makes it; for
// judgement calls on their words (tighten, improve, reword) it suggests,
// deciding case by case: these words in place of those, and why,
// each waiting on the record's page for whoever may change it to accept
// or decline. Accepting is an ordinary change, logged and undone like any
// other; a suggestion whose words have changed since says so rather than
// guessing where it was meant to go.

type suggested struct {
	Passage     string `json:"passage"`
	Replacement string `json:"replacement"`
	Why         string `json:"why"`
	Kind        string `json:"kind"`
	Meaning     bool   `json:"meaning"`
}

// mostSuggested is how many changes one pass suggests: a writer handed
// hundreds stops reading them, and an editor picks what matters.
const mostSuggested = 15

func (s *Service) suggestTools() []llm.Tool {
	if _, ok := s.Store.Types().Get(SuggestionType); !ok {
		return nil
	}
	return []llm.Tool{{Name: "suggest_edits",
		Description: "Suggest changes to a person's writing instead of making them; they accept or decline each on its page. Judge each case: a plain command (fix the spelling) you just do with update_record; suggest when the changes are judgement calls on their words. Do only the help asked for, keep their voice, small changes, at most 15. Feedback on structure is said in words, not suggested. Copy each passage exactly from get_record, long enough to occur once.",
		Schema: obj(map[string]any{
			"type":  map[string]any{"type": "string", "description": "The record's content type, such as note."},
			"id":    map[string]any{"type": "string", "description": "The record's id."},
			"field": map[string]any{"type": "string", "description": "The field; its main text when left out."},
			"edits": map[string]any{"type": "array", "items": obj(map[string]any{
				"passage":     map[string]any{"type": "string", "description": "The words now, exactly."},
				"replacement": map[string]any{"type": "string", "description": "What replaces them; empty takes them out."},
				"why":         map[string]any{"type": "string", "description": "Why, in a short sentence."},
				"kind":        map[string]any{"type": "string", "enum": []string{"fix", "clarity", "style", "structure"}, "description": "fix: spelling, grammar, typos; clarity; style; structure: moving or cutting."},
				"meaning":     map[string]any{"type": "boolean", "description": "It changes what is said, not only how."},
			}, "passage", "replacement", "why", "kind")},
		}, "type", "id", "edits")}}
}

// textField is the field a suggestion changes: the one named, or the
// record's main text (its first markdown field, else its first text one).
func textField(t *schema.Type, name string) (string, error) {
	if name != "" {
		f, ok := t.Field(name)
		if !ok || f.ReadOnly || f.Type != "markdown" && f.Type != "text" && f.Type != "string" {
			return "", fmt.Errorf("%s has no field %q of words that can be changed", t.Name, name)
		}
		return name, nil
	}
	for _, kind := range []string{"markdown", "text"} {
		for _, f := range t.Fields {
			if f.Type == kind && !f.ReadOnly {
				return f.Name, nil
			}
		}
	}
	return "", fmt.Errorf("%s has no field of writing; name the field", t.Name)
}

func (s *Service) suggestEdits(typeName, id, field string, edits []suggested) toolResult {
	t, err := records.ContentType(s.Store, typeName)
	if err != nil {
		return fail("%v", err)
	}
	rec, err := s.Store.Get(t.Name, id)
	if err != nil {
		return fail("no %s %s; find_records to get its id", t.Name, id)
	}
	if field, err = textField(t, field); err != nil {
		return fail("%v", err)
	}
	if len(edits) == 0 {
		return fail("no edits given; each is a passage, its replacement and why")
	}
	if len(edits) > mostSuggested {
		return fail("nothing suggested: %d changes is more than a writer can weigh in one go; suggest the %d that matter most", len(edits), mostSuggested)
	}
	text := wordsOf(rec.Fields[field])
	// Each passage is found once, and none overlaps another, so each can
	// be accepted on its own in any order.
	var problems []string
	type span struct{ from, to int }
	var spans []span
	for i, e := range edits {
		n := strings.Count(text, e.Passage)
		switch {
		case e.Passage == "":
			problems = append(problems, fmt.Sprintf("edit %d has no passage", i+1))
		case n == 0:
			problems = append(problems, fmt.Sprintf("edit %d: %q is not in the %s; copy it exactly from get_record", i+1, trim.Clip(e.Passage, 60), field))
		case n > 1:
			problems = append(problems, fmt.Sprintf("edit %d: %q occurs %d times; give more of the words around it so it occurs once", i+1, trim.Clip(e.Passage, 60), n))
		case e.Passage == e.Replacement:
			problems = append(problems, fmt.Sprintf("edit %d changes nothing", i+1))
		default:
			from := strings.Index(text, e.Passage)
			for _, sp := range spans {
				if from < sp.to && sp.from < from+len(e.Passage) {
					problems = append(problems, fmt.Sprintf("edit %d overlaps another; make them one edit", i+1))
				}
			}
			spans = append(spans, span{from, from + len(e.Passage)})
		}
	}
	if len(problems) > 0 {
		return fail("nothing suggested: %s", strings.Join(problems, "; "))
	}
	about := t.Name + "/" + rec.ID
	var batch []BatchItem
	for _, e := range edits {
		kind := e.Kind
		if _, only := prose.FormatChange(e.Passage, e.Replacement); only {
			kind = "format"
		}
		made, err := s.Store.Create(SuggestionType, map[string]any{"about": about, "field": field, "kind": kind, "meaning": e.Meaning,
			"passage": e.Passage, "replacement": e.Replacement, "why": strings.TrimSpace(e.Why)})
		if err != nil {
			return fail("could not suggest: %v", err)
		}
		batch = append(batch, BatchItem{Type: SuggestionType, ID: made.ID})
	}
	title := Name(s.Store, t, rec)
	page := "/t/" + t.Name + "/" + rec.ID
	c := Change{Action: "suggested", Component: t.Name, ID: rec.ID, Href: page,
		Detail: fmt.Sprintf("%s, %s", title, schema.Count(len(edits), "change")), Before: Batch(batch)}
	return toolResult{text: fmt.Sprintf("suggested %s to %s %s; they wait on its page, %s, for the person to accept or decline each. Nothing is changed until they do.",
		schema.Count(len(edits), "change"), t.Name, title, page), change: &c}
}

// Suggestions are those waiting on a record, oldest first.
func Suggestions(st *store.Store, typeName, id string) []*store.Record {
	if _, ok := st.Types().Get(SuggestionType); !ok {
		return nil
	}
	all, _ := st.List(SuggestionType, store.ListOptions{OrderBy: "created_at"})
	var out []*store.Record
	for _, r := range all {
		if r.Fields["about"] == typeName+"/"+id && r.Fields["state"] == "pending" {
			out = append(out, r)
		}
	}
	return out
}

// ErrOutdated is a suggestion whose words are no longer there.
var ErrOutdated = errors.New("the words it would change have changed since, so it no longer fits; it is set aside")

// AcceptSuggestions makes suggested changes to one record, as who's, in
// one change to it: one entry in the log, one Undo. One whose words have
// changed since is set aside, not guessed at. It returns the record, its
// entry, how many went in and how many no longer fitted.
func AcceptSuggestions(st *store.Store, who Who, ids []string) (rec *store.Record, entry string, made, outdated int, err error) {
	var typ, recID string
	fields := map[string]any{}
	var took []string
	for _, id := range ids {
		sg, err := st.Get(SuggestionType, id)
		if err != nil || sg.Fields["state"] != "pending" {
			continue
		}
		t, r, _ := strings.Cut(wordsOf(sg.Fields["about"]), "/")
		if typ == "" {
			typ, recID = t, r
			if rec, err = st.Get(typ, recID); err != nil {
				DeclineSuggestion(st, id)
				return nil, "", 0, 0, errors.New("what it was about is gone")
			}
		}
		if t != typ || r != recID {
			continue // one record at a time
		}
		field := wordsOf(sg.Fields["field"])
		text, ok := fields[field].(string)
		if !ok {
			text = wordsOf(rec.Fields[field])
		}
		passage := wordsOf(sg.Fields["passage"])
		if passage == "" || strings.Count(text, passage) != 1 {
			DeclineSuggestion(st, id)
			outdated++
			continue
		}
		fields[field] = strings.Replace(text, passage, wordsOf(sg.Fields["replacement"]), 1)
		took = append(took, id)
	}
	if len(took) == 0 {
		if outdated > 0 {
			return rec, "", 0, outdated, ErrOutdated
		}
		return nil, "", 0, 0, errors.New("that suggestion is not waiting any more")
	}
	if rec, entry, err = WriteAs(st, who, "updated", typ, recID, fields); err != nil {
		return nil, "", 0, outdated, err
	}
	for _, id := range took {
		st.Update(SuggestionType, id, map[string]any{"state": "accepted"})
	}
	return rec, entry, len(took), outdated, nil
}

// DeclineSuggestion sets a suggestion aside, changing nothing else.
func DeclineSuggestion(st *store.Store, id string) error {
	_, err := st.Update(SuggestionType, id, map[string]any{"state": "declined"})
	return err
}

// wordsOf is a field's words, or "" for none.
func wordsOf(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
