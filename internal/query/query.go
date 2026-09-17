// Package query is one small grammar for asking which records: the same
// words in a collection block on the canvas, in a list page's address, in
// the assistant's find_records, on the command line and over the API, so
// a view a person makes and a view the model makes are the same view.
package query

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Grammar is the whole language, written once and shown wherever a query
// can be typed.
const Grammar = "Each condition is field, operator, value with no spaces around the operator: " +
	"status=draft, status!=done, title~garden (contains), due<today, due>=+7d, tags=health (has), " +
	"body= (empty), due!= (set). Dates take 2026-10-01, today, tomorrow, yesterday, now, +7d, -1w, +3h. " +
	"Order is a field name, or -field for the largest or newest first; created_at and updated_at work too."

// Cond is one condition.
type Cond struct {
	Field, Op, Value string
}

func (c Cond) String() string { return c.Field + c.Op + c.Value }

var ops = []string{"!=", "<=", ">=", "~", "<", ">", "="}

// Parse reads one condition against a type, so a wrong field is a clear
// error naming the fields there are.
func Parse(t *schema.Type, s string) (Cond, error) {
	s = strings.TrimSpace(s)
	at, op := -1, ""
	for _, o := range ops {
		if i := strings.Index(s, o); i > 0 && (at < 0 || i < at) {
			at, op = i, o
		}
	}
	if at < 0 {
		return Cond{}, fmt.Errorf("%q is not a condition. %s", s, Grammar)
	}
	c := Cond{Field: strings.TrimSpace(s[:at]), Op: op, Value: strings.TrimSpace(s[at+len(op):])}
	if !known(t, c.Field) {
		return Cond{}, fmt.Errorf("%s has no field %q; it has %s", t.Name, c.Field, strings.Join(fieldNames(t), ", "))
	}
	return c, nil
}

// ParseAll reads every condition or says which one is wrong.
func ParseAll(t *schema.Type, where []string) ([]Cond, error) {
	var out []Cond
	for _, w := range where {
		if strings.TrimSpace(w) == "" {
			continue
		}
		c, err := Parse(t, w)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// Filter lists the records of a type that match every condition, in the
// order asked for, at most limit of them. now anchors relative dates.
func Filter(st *store.Store, t *schema.Type, where []string, order string, limit int, now time.Time) ([]*store.Record, error) {
	conds, err := ParseAll(t, where)
	if err != nil {
		return nil, err
	}
	field := strings.TrimPrefix(order, "-")
	if field != "" && !known(t, field) {
		return nil, fmt.Errorf("%s has no field %q to order by; it has %s", t.Name, field, strings.Join(fieldNames(t), ", "))
	}
	recs, err := st.List(t.Name, store.ListOptions{})
	if err != nil {
		return nil, err
	}
	// A ref is matched by the id it holds or by the title of what it
	// points at, read once per target.
	titles := map[string]string{}
	lookup := func(to, id string) string {
		key := to + "/" + id
		if title, ok := titles[key]; ok {
			return title
		}
		title := ""
		if target, err := st.Get(to, id); err == nil {
			if tt, ok := st.Types().Get(to); ok {
				if v, ok := target.Fields[tt.Title].(string); ok {
					title = v
				}
			}
		}
		titles[key] = title
		return title
	}
	var out []*store.Record
	for _, rec := range recs {
		if matchWith(t, rec, conds, now, lookup) {
			out = append(out, rec)
		}
	}
	if field != "" {
		Sort(t, out, order)
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Match says whether a record meets every condition. A ref field matches
// on the id it holds; Filter also matches it on the target's title.
func Match(t *schema.Type, rec *store.Record, conds []Cond, now time.Time) bool {
	return matchWith(t, rec, conds, now, nil)
}

func matchWith(t *schema.Type, rec *store.Record, conds []Cond, now time.Time, lookup func(to, id string) string) bool {
	for _, c := range conds {
		if !matchOne(t, rec, c, now, lookup) {
			return false
		}
	}
	return true
}

func matchOne(t *schema.Type, rec *store.Record, c Cond, now time.Time, lookup func(to, id string) string) bool {
	kind := kindOf(t, c.Field)
	v := valueOf(rec, c.Field)
	if kind == "ref" && c.Value != "" {
		id, _ := v.(string)
		if id == c.Value {
			return c.Op == "=" || c.Op == "~"
		}
		if lookup != nil && id != "" {
			if f, ok := t.Field(c.Field); ok {
				return compare("string", lookup(f.To, id), c.Op, c.Value, now)
			}
		}
		return c.Op == "!="
	}
	if c.Value == "" {
		switch c.Op {
		case "=":
			return empty(v)
		case "!=":
			return !empty(v)
		}
	}
	if kind == "list" {
		items, _ := v.([]any)
		for _, it := range items {
			if compare("string", it, c.Op, c.Value, now) {
				return true
			}
		}
		return false
	}
	return compare(kind, v, c.Op, c.Value, now)
}

// compare is one typed comparison: dates as instants, numbers as numbers,
// words without regard to case.
func compare(kind string, v any, op, want string, now time.Time) bool {
	switch kind {
	case "datetime":
		have, ok1 := v.(string)
		ht, err := time.Parse(time.RFC3339, have)
		wt, day, ok2 := parseDate(want, now)
		if !ok1 || err != nil || !ok2 {
			return false
		}
		if day && (op == "=" || op == "!=") {
			same := ht.Local().Format("2006-01-02") == wt.Local().Format("2006-01-02")
			return same == (op == "=")
		}
		return ordered(ht.Compare(wt), op)
	case "int", "float":
		have, ok1 := number(v)
		wantN, ok2 := number(want)
		if !ok1 || !ok2 {
			return false
		}
		switch {
		case have < wantN:
			return ordered(-1, op)
		case have > wantN:
			return ordered(1, op)
		}
		return ordered(0, op)
	case "bool":
		have, _ := v.(bool)
		wantB := want == "true" || want == "yes" || want == "1"
		return (have == wantB) == (op == "=" || op == "<=" || op == ">=")
	}
	have := strings.ToLower(text(v))
	want = strings.ToLower(want)
	switch op {
	case "~":
		return strings.Contains(have, want)
	case "=":
		return have == want
	case "!=":
		return have != want
	}
	return ordered(strings.Compare(have, want), op)
}

func ordered(cmp int, op string) bool {
	switch op {
	case "=":
		return cmp == 0
	case "!=":
		return cmp != 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	}
	return false
}

// Sort orders records by a field, "-field" for the largest or newest
// first; created_at and updated_at are the record's own times.
func Sort(t *schema.Type, recs []*store.Record, order string) {
	desc := strings.HasPrefix(order, "-")
	field := strings.TrimPrefix(order, "-")
	kind := kindOf(t, field)
	sort.SliceStable(recs, func(i, j int) bool {
		less := lessThan(kind, valueOf(recs[i], field), valueOf(recs[j], field))
		if desc {
			return lessThan(kind, valueOf(recs[j], field), valueOf(recs[i], field))
		}
		return less
	})
}

func lessThan(kind string, a, b any) bool {
	switch kind {
	case "int", "float":
		x, _ := number(a)
		y, _ := number(b)
		return x < y
	case "bool":
		x, _ := a.(bool)
		y, _ := b.(bool)
		return !x && y
	}
	// Dates are RFC 3339 strings, which sort as they should.
	return strings.ToLower(text(a)) < strings.ToLower(text(b))
}
