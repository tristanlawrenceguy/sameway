package schema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/fold"
)

// What a refused field takes. Refused, a write said "at: is required" and
// nothing of what at is: an agent logging a run had to guess a moment's
// shape, and a field it wrote under another name ("notes" for "body") was
// "unknown field" with no word of what the type does take (crew finding
// 0741). A refusal now says, for each field it names, what goes there,
// with an example; for one the type does not have, the nearest it does,
// and every field it takes.

// Expect is what a field takes, in words a writer can act on.
func Expect(f Field) string {
	what := f.Words()
	switch {
	case f.RefList():
		return fmt.Sprintf("%s: the ids of %s records, as [\"id1\", \"id2\"], from GET /api/%s", what, f.To, f.To)
	case f.Type == "ref":
		return fmt.Sprintf("%s: the id of a %s record, from GET /api/%s", what, f.To, f.To)
	case f.Type == "enum":
		return fmt.Sprintf("%s: one of %s", what, strings.Join(f.Values, ", "))
	case f.Type == "datetime":
		return what + ": a moment in the person's own time, as 2026-10-14T09:00 or words such as today 9am or next Friday 2pm, or a day as 2026-10-14; a time ending in Z is UTC, not theirs"
	case f.Type == "repeat":
		return what + ": how often, in words such as every Monday or every day"
	case f.Type == "bool":
		return what + ": true or false"
	case f.Type == "int":
		return what + ": a whole number"
	case f.Type == "float":
		return what + ": a number"
	case f.Type == "list":
		return fmt.Sprintf("%s: a list of %s, as [\"a\", \"b\"]", what, f.Of)
	case f.MaxLength > 0:
		return fmt.Sprintf("%s: words, at most %d characters", what, f.MaxLength)
	}
	return what + ": words"
}

// Expects is what each refused field of t takes, by its name: a field t
// has, what goes there; one it does not, the nearest it has and all it
// takes.
func (t *Type) Expects(problems map[string]string) map[string]string {
	out := map[string]string{}
	for name := range problems {
		if f, ok := t.Field(name); ok {
			out[name] = Expect(*f)
			continue
		}
		taken := t.writable()
		said := fmt.Sprintf("%s has no field %s; it takes %s", t.Name, name, strings.Join(taken, ", "))
		if near := fold.Nearest(name, taken); near != "" {
			said = fmt.Sprintf("%s has no field %s: did you mean %s? It takes %s", t.Name, name, near, strings.Join(taken, ", "))
		}
		out[name] = said
	}
	return out
}

// writable is the names of the fields a writer gives.
func (t *Type) writable() []string {
	var out []string
	for _, f := range t.Fields {
		if !f.Hidden && !f.ReadOnly && !ServerFilled(f.Description) {
			out = append(out, f.Name)
		}
	}
	sort.Strings(out)
	return out
}
