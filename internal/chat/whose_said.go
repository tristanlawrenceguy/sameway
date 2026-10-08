package chat

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Turning a meeting's notes into tasks "for who said they'd do what", a
// small model wrote "Ana Silva: send new price list" and left the task's
// for empty, though Ana Silva was a person in the workspace: the task was
// hers only in its words. So a record made with a person's name in its
// words, and the field that says whose it is left empty, says so, with
// the person's id to set.

// whoseSaid is that note, or "".
func (s *Service) whoseSaid(t *schema.Type, rec *store.Record) string {
	var field string
	for _, f := range t.Fields {
		if f.Type == "ref" && f.To == PersonType {
			if v, _ := rec.Fields[f.Name].(string); v != "" {
				return ""
			}
			if field == "" {
				field = f.Name
			}
		}
	}
	if field == "" {
		return ""
	}
	if _, ok := s.Store.Types().Get(PersonType); !ok {
		return ""
	}
	people, err := s.Store.List(PersonType, store.ListOptions{})
	if err != nil {
		return ""
	}
	title, body := recordTitle(s.Store, t, rec), ""
	for _, f := range t.Fields {
		if v, ok := rec.Fields[f.Name].(string); ok && (f.Type == "text" || f.Type == "markdown" || f.Type == "string") {
			body += " " + v
		}
	}
	words := nameWords(title + " " + body)
	var found []string
	for _, p := range people {
		name, _ := p.Fields["name"].(string)
		if name != "" && namedIn(name, words) {
			found = append(found, fmt.Sprintf("%s (%s %s)", name, PersonType, p.ID))
		}
	}
	if len(found) != 1 {
		return "" // none, or several: whose it is is for the model to say
	}
	return fmt.Sprintf(" Its words name %s, but its %s is empty: if it is theirs, set %s to that id with update_record.", found[0], field, field)
}

func nameWords(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && r != '\'' }) {
		out[strings.TrimSuffix(w, "'s")] = true
	}
	return out
}

// namedIn is whether a person's whole name, or their first name of three
// letters or more, is among the words.
func namedIn(name string, words map[string]bool) bool {
	parts := strings.Fields(strings.ToLower(name))
	if len(parts) == 0 {
		return false
	}
	all := true
	for _, p := range parts {
		all = all && words[p]
	}
	return all || len([]rune(parts[0])) >= 3 && words[parts[0]]
}
