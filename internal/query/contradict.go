package query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Contradiction says when conditions can never all hold, because they ask
// one field to equal two different values: status=reading and
// status=to read, which no record meets, since a record has one status.
// Every condition holds at once, so such a view is empty for good, not
// only for now. A list field (tags=a and tags=b is a record with both), a
// date (today and 2026-10-01 may be one day) and a ref (an id and a title
// may name one record) can hold two at once and are left alone. It is
// empty when nothing contradicts, or when a condition does not parse,
// which Filter says itself.
func Contradiction(t *schema.Type, where []string) string {
	conds, err := ParseAll(t, where)
	if err != nil {
		return ""
	}
	first := map[string]Cond{}
	for _, c := range conds {
		if c.Op != "=" {
			continue
		}
		switch kindOf(t, c.Field) {
		case "list", "datetime", "ref":
			continue
		}
		prev, seen := first[c.Field]
		if !seen {
			first[c.Field] = c
			continue
		}
		if same(kindOf(t, c.Field), prev.Value, c.Value) {
			continue
		}
		return fmt.Sprintf("%s and %s can never both hold, since a %s has one %s, so it would never show anything; use one value, or one block per value (one list per %s)",
			prev, c, schema.Words(t.Name), fieldWord(t, c.Field), fieldWord(t, c.Field))
	}
	return ""
}

// same is whether two values a field is asked to equal are one value, as
// the comparison reads them.
func same(kind, a, b string) bool {
	a, b = strings.ToLower(strings.TrimSpace(a)), strings.ToLower(strings.TrimSpace(b))
	switch kind {
	case "bool":
		yes := func(s string) bool { return s == "true" || s == "yes" || s == "1" }
		return yes(a) == yes(b)
	case "int", "float":
		x, err1 := strconv.ParseFloat(a, 64)
		y, err2 := strconv.ParseFloat(b, 64)
		if err1 == nil && err2 == nil {
			return x == y
		}
	}
	return a == b
}

func fieldWord(t *schema.Type, field string) string {
	if f, ok := t.Field(field); ok && f.Label != "" {
		return strings.ToLower(f.Label)
	}
	return strings.ReplaceAll(field, "_", " ")
}
