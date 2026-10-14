package schema

import (
	"fmt"
	"sort"
	"strings"
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
		return what + ": a moment, as 2026-10-14T09:00:00Z, or a day as 2026-10-14, or words such as today 9am or next Friday"
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
		if near := nearest(name, taken); near != "" {
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

// nearest is the name in names most like name, when one is close: a slip
// of a letter or two, or one the other holds (note for notes).
func nearest(name string, names []string) string {
	best, bestD := "", 3
	for _, n := range names {
		if strings.Contains(n, name) || strings.Contains(name, n) {
			return n
		}
		if d := distance(name, n); d < bestD {
			best, bestD = n, d
		}
	}
	return best
}

// distance is how many letters must change to make a into b.
func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
