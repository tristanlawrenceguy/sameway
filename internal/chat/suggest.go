package chat

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Someone's own words are theirs to change. Asked to improve them, the
// assistant or an agent suggests: these words in place of those, and why,
// each waiting on the record's page for whoever may change it to accept
// or decline. Accepting is an ordinary change, logged and undone like any
// other; a suggestion whose words have changed since says so rather than
// guessing where it was meant to go.

// SuggestionType is the content type of a suggested change.
const SuggestionType = "suggestion"

type suggested struct {
	Passage     string `json:"passage"`
	Replacement string `json:"replacement"`
	Why         string `json:"why"`
}

func (s *Service) suggestTools() []llm.Tool {
	if _, ok := s.Store.Types().Get(SuggestionType); !ok {
		return nil
	}
	return []llm.Tool{{Name: "suggest_edits",
		Description: "Suggest changes to someone's writing instead of making them: each is a passage as it is now, the words to put in its place and why, shown on the record's page for the person to accept or decline one by one. Use it whenever you are asked to improve, shorten, correct, translate or tidy words a person wrote; use update_record only for what is yours to write. Read the record first with get_record, and copy each passage exactly, with enough words that it occurs once.",
		Schema: obj(map[string]any{
			"type":  map[string]any{"type": "string", "description": "The record's content type, such as note."},
			"id":    map[string]any{"type": "string", "description": "The record's id."},
			"field": map[string]any{"type": "string", "description": "The field whose words change; the record's main text when left out."},
			"edits": map[string]any{"type": "array", "description": "The changes, each to a different passage.", "items": obj(map[string]any{
				"passage":     map[string]any{"type": "string", "description": "The words as they are now, exactly, occurring once."},
				"replacement": map[string]any{"type": "string", "description": "The words to put in their place; empty takes the passage out."},
				"why":         map[string]any{"type": "string", "description": "Why, in a short sentence the person reads to decide."},
			}, "passage", "replacement", "why")},
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
	t, err := s.contentType(typeName)
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
			problems = append(problems, fmt.Sprintf("edit %d: %q is not in the %s; copy it exactly from get_record", i+1, clip(e.Passage, 60), field))
		case n > 1:
			problems = append(problems, fmt.Sprintf("edit %d: %q occurs %d times; give more of the words around it so it occurs once", i+1, clip(e.Passage, 60), n))
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
		made, err := s.Store.Create(SuggestionType, map[string]any{"about": about, "field": field,
			"passage": e.Passage, "replacement": e.Replacement, "why": strings.TrimSpace(e.Why)})
		if err != nil {
			return fail("could not suggest: %v", err)
		}
		batch = append(batch, BatchItem{Type: SuggestionType, ID: made.ID})
	}
	title := Name(s.Store, t, rec)
	page := "/t/" + t.Name + "/" + rec.ID
	c := Change{Action: "suggested", Component: t.Name, ID: rec.ID, Href: page,
		Detail: fmt.Sprintf("%s, %s", title, count(len(edits), "change")), Before: Batch(batch)}
	return toolResult{text: fmt.Sprintf("suggested %s to %s %s; they wait on its page, %s, for the person to accept or decline each. Nothing is changed until they do.",
		count(len(edits), "change"), t.Name, title, page), change: &c}
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

// AcceptSuggestion makes a suggested change, as who's, and returns the
// record it changed and its entry in the log.
func AcceptSuggestion(st *store.Store, who Who, id string) (*store.Record, string, error) {
	sg, err := st.Get(SuggestionType, id)
	if err != nil || sg.Fields["state"] != "pending" {
		return nil, "", errors.New("that suggestion is not waiting any more")
	}
	typ, recID, _ := strings.Cut(wordsOf(sg.Fields["about"]), "/")
	field := wordsOf(sg.Fields["field"])
	rec, err := st.Get(typ, recID)
	if err != nil {
		DeclineSuggestion(st, id)
		return nil, "", errors.New("what it was about is gone")
	}
	text, passage := wordsOf(rec.Fields[field]), wordsOf(sg.Fields["passage"])
	if strings.Count(text, passage) != 1 {
		DeclineSuggestion(st, id)
		return nil, "", ErrOutdated
	}
	changed, entry, err := WriteAs(st, who, "updated", typ, recID, map[string]any{field: strings.Replace(text, passage, wordsOf(sg.Fields["replacement"]), 1)})
	if err != nil {
		return nil, "", err
	}
	st.Update(SuggestionType, id, map[string]any{"state": "accepted"})
	return changed, entry, nil
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
