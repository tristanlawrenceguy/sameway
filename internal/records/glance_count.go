package records

import (
	"fmt"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What is in a record, at a glance (design/foundations/glance.md): how
// many records of another type are in it, and how many of those are done
// when they can be: a project's "3 tasks, 1 done", a person's "4
// interactions". One rule, from the schema, says what a record holds, for
// its glance and its page alike: a ref marked listed (task.project,
// task.event, interaction.person in the starter) puts its records in the
// one it points at, counted here and listed on that record's page
// (backrefs.go). Any other ref (a task's For) says who or what it
// concerns, not what it is in. None is not said, as a state where every
// record rests is not.
//
// A listing counts once for all its rows, one query for each type that
// can be in its records, never one for each row.

// Counts is what is in each record of a listing, by its id, in words.
type Counts map[string][]string

// Within is a type whose records can be in a record of another, and the
// field that says which.
type Within struct {
	T     *schema.Type
	Field string
}

// Withins are the types whose records can be in a record of t.
func Withins(st *store.Store, t *schema.Type) []Within {
	var out []Within
	for _, u := range st.Types().Types {
		if u.Internal || u.Name == t.Name {
			continue
		}
		for _, f := range u.Shown() {
			if f.Listed && f.Type == "ref" && f.To == t.Name {
				out = append(out, Within{u, f.Name})
			}
		}
	}
	return out
}

// CountsOf counts what is in each of recs, all of t: one query for each
// type that can be in them, for the whole listing at once.
func CountsOf(st *store.Store, t *schema.Type, recs []*store.Record) Counts {
	out := Counts{}
	if len(recs) == 0 {
		return out
	}
	for _, w := range Withins(st, t) {
		cond := w.Field + "!=" // set: every one in something, tallied below
		if len(recs) == 1 {
			cond = w.Field + "=" + recs[0].ID
		}
		in, err := query.Filter(st, w.T, []string{cond}, "", 0, time.Now())
		if err != nil {
			continue
		}
		done := w.T.DoneField()
		n, ticked := map[string]int{}, map[string]int{}
		for _, r := range in {
			id, _ := r.Fields[w.Field].(string)
			n[id]++
			if done != nil {
				if on, _ := r.Fields[done.Name].(bool); on {
					ticked[id]++
				}
			}
		}
		for _, rec := range recs {
			if n[rec.ID] == 0 {
				continue
			}
			words := schema.Count(n[rec.ID], w.T.Name)
			if ticked[rec.ID] > 0 {
				words += ", " + DoneWords(done, ticked[rec.ID])
			}
			out[rec.ID] = append(out[rec.ID], words)
		}
	}
	return out
}

// DoneWords is how many are ticked, by the tick's own word: 1 done.
func DoneWords(f *schema.Field, n int) string {
	return fmt.Sprintf("%d %s", n, f.Words())
}
