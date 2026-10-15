package query

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A condition on a ref holds the other record's id, and was said with it:
// a habit's entries read "habit is ylaohctaosdr4dyd", a string a person
// cannot place. WordsIn says it by the record's title, read from the
// store, "habit is Running miles"; an id it cannot find stays as written.

// WordsIn is Words with each record a condition names said by its title.
func WordsIn(st *store.Store, t *schema.Type, where []string, loc *time.Location) string {
	var out []string
	for _, w := range where {
		if strings.TrimSpace(w) == "" {
			continue
		}
		c, err := Parse(t, w)
		if err != nil {
			out = append(out, w)
			continue
		}
		if kindOf(t, c.Field) == "ref" && c.Value != "" {
			if title := titleOf(st, t, c.Field, c.Value); title != "" {
				c.Value = title
			}
		}
		out = append(out, condWords(t, c, loc))
	}
	return schema.AndNames(out) // a value can have a comma of its own
}

// titleOf is the title of the record a ref field names, or "".
func titleOf(st *store.Store, t *schema.Type, field, id string) string {
	f, ok := t.Field(field)
	if !ok || f.To == "" || st == nil {
		return ""
	}
	to, ok := st.Types().Get(f.To)
	if !ok {
		return ""
	}
	rec, err := st.Get(f.To, id)
	if err != nil {
		return ""
	}
	title, _ := rec.Fields[to.Title].(string)
	return title
}
