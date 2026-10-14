package query

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Why a list could not be read as asked, as a value a page can say in a
// person's words. The words an agent is given name fields as they are
// stored (starts, ends, uid, created_at), right for writing the next
// request and wrong on a person's screen, where a list page showed them
// (crew, task 0279). Error is the agent's; a page asks what it is.

// Refused is a condition or an order a list cannot take.
type Refused struct {
	Type  *schema.Type
	Field string // the field asked for, when there is one
	Text  string // the condition as written, when it is not one
	Order bool   // asked to order by Field, not to narrow by it
}

func (r *Refused) Error() string {
	switch {
	case r.Field == "":
		return fmt.Sprintf("%q is not a condition. %s", r.Text, Grammar)
	case r.Order:
		return fmt.Sprintf("%s has no field %q to order by; it has %s", r.Type.Name, r.Field, strings.Join(fieldNames(r.Type), ", "))
	}
	return fmt.Sprintf("%s has no field %q; it has %s", r.Type.Name, r.Field, strings.Join(fieldNames(r.Type), ", "))
}

// Narrowable is what a person can narrow or order the type's list by, in
// the words its pages use.
func Narrowable(t *schema.Type) []string {
	var out []string
	for _, f := range t.Shown() {
		out = append(out, f.Display())
	}
	return append(out, "when added", "when last changed")
}
