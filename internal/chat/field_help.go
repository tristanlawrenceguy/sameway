package chat

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/fold"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// fieldHelp is what a refused write is told, once each: every field that
// was wrong and how to put it right, in the assistant's own tools (a ref's
// id from find_records, not an address of the API), then every field the
// type takes. Said twice ("status: must be one of todo, doing, done.
// status: one of todo, doing, done"), and with the fields listed twice, a
// refusal was long to read for what it had to say.
func fieldHelp(err error, t *schema.Type) string {
	var ve *schema.ValidationError
	if !errors.As(err, &ve) || len(ve.Problems) == 0 {
		return humanizeValidationError(err.Error()) + ". " + typeHelp(t)
	}
	var said []string
	for name, problem := range ve.Problems {
		said = append(said, fieldWrong(t, name, problem))
	}
	sort.Strings(said)
	return strings.Join(said, "; ") + ". " + typeHelp(t)
}

// fieldWrong is one field that was wrong, and what goes there.
func fieldWrong(t *schema.Type, name, problem string) string {
	f, ok := t.Field(name)
	if !ok {
		said := t.Name + " has no field " + name
		if near := fold.Nearest(name, typeFieldNames(t)); near != "" {
			said += " (did you mean " + near + "?)"
		}
		return said + "; to keep a new one, add_field first"
	}
	switch {
	case f.Type == "ref" || f.RefList():
		return fmt.Sprintf("%s %s: the id of a %s, from find_records on %s", name, problem, f.To, f.To)
	case f.Type == "datetime" && strings.Contains(problem, "required"):
		return name + " is required: a day or time, in words such as tomorrow 2pm"
	case strings.Contains(problem, "required"):
		return name + " is required"
	}
	return name + ": " + problem
}

func typeFieldNames(t *schema.Type) []string {
	var out []string
	for _, f := range t.Fields {
		out = append(out, f.Name)
	}
	return out
}
