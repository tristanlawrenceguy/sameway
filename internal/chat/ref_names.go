package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A field that holds several records, such as a meeting's people, is
// written as ids by tools and agents and as names by a person, who types
// "Ann Lee, Ben Ortiz". Every write comes through here, so each name is
// found once, for every way in: by an id, by its title, or for a person by
// their email. A name that is not anyone, or is two people, is refused
// with what to do; nobody is made up from a typo.

func refNamesToIDs(st *store.Store, t *schema.Type, fields map[string]any) error {
	problems := map[string]string{}
	for _, f := range t.Fields {
		raw, sent := fields[f.Name]
		if !sent || !f.RefList() {
			continue
		}
		items, err := listItems(raw)
		if err != nil {
			problems[f.Name] = err.Error()
			continue
		}
		to, ok := st.Types().Get(f.To)
		if !ok {
			continue
		}
		ids := make([]any, 0, len(items))
		for _, item := range items {
			id, why := findByName(st, to, item)
			if why != "" {
				problems[f.Name] = why
				break
			}
			if !containsID(ids, id) {
				ids = append(ids, id)
			}
		}
		fields[f.Name] = ids
	}
	if len(problems) > 0 {
		return &schema.ValidationError{Problems: problems}
	}
	return nil
}

func listItems(raw any) ([]string, error) {
	var out []string
	switch l := raw.(type) {
	case nil:
	case []any:
		for _, v := range l {
			out = append(out, strings.TrimSpace(fmt.Sprint(v)))
		}
	case []string:
		out = append(out, l...)
	case string:
		if strings.HasPrefix(strings.TrimSpace(l), "[") {
			var arr []any
			if err := json.Unmarshal([]byte(l), &arr); err != nil {
				return nil, fmt.Errorf("must be a JSON array or names separated by commas")
			}
			return listItems(arr)
		}
		for _, part := range strings.FieldsFunc(l, func(r rune) bool { return r == ',' || r == '\n' || r == ';' }) {
			out = append(out, strings.TrimSpace(part))
		}
	default:
		return nil, fmt.Errorf("must be a list")
	}
	kept := out[:0]
	for _, s := range out {
		if s != "" {
			kept = append(kept, s)
		}
	}
	return kept, nil
}

// findByName is the record a name means: its id, its title, or a
// person's email. why says what is wrong when it means none or several.
func findByName(st *store.Store, t *schema.Type, name string) (id, why string) {
	if _, err := st.Get(t.Name, name); err == nil {
		return name, ""
	}
	all, _ := st.List(t.Name, store.ListOptions{})
	var found []string
	for _, r := range all {
		email, _ := r.Fields["email"].(string)
		if strings.EqualFold(Name(st, t, r), name) || email != "" && strings.EqualFold(email, name) {
			found = append(found, r.ID)
		}
	}
	switch len(found) {
	case 1:
		return found[0], ""
	case 0:
		return "", fmt.Sprintf("no %s is called %q; check the spelling, or add them as a %s first", t.Name, name, t.Name)
	}
	return "", fmt.Sprintf("%d %s records are called %q; give the email or pick one by its id", len(found), t.Name, name)
}

func containsID(ids []any, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
