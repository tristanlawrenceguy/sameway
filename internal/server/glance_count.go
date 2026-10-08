package server

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

// counts is what is in each record of a listing, by its id, in words.
type counts map[string][]string

// within is a type whose records can be in a record of another, and the
// field that says which.
type within struct {
	t     *schema.Type
	field string
}

// withins are the types whose records can be in a record of t.
func (s *Server) withins(t *schema.Type) []within {
	var out []within
	for _, u := range s.app.Types.Types {
		if u.Internal || u.Name == t.Name {
			continue
		}
		for _, f := range u.Shown() {
			if f.Listed && f.Type == "ref" && f.To == t.Name {
				out = append(out, within{u, f.Name})
			}
		}
	}
	return out
}

// countsOf counts what is in each of recs, all of t: one query for each
// type that can be in them, for the whole listing at once.
func (s *Server) countsOf(t *schema.Type, recs []*store.Record) counts {
	out := counts{}
	if len(recs) == 0 {
		return out
	}
	for _, w := range s.withins(t) {
		cond := w.field + "!=" // set: every one in something, tallied below
		if len(recs) == 1 {
			cond = w.field + "=" + recs[0].ID
		}
		in, err := query.Filter(s.app.Store, w.t, []string{cond}, "", 0, time.Now())
		if err != nil {
			continue
		}
		done := w.t.DoneField()
		n, ticked := map[string]int{}, map[string]int{}
		for _, r := range in {
			id, _ := r.Fields[w.field].(string)
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
			words := schema.Count(n[rec.ID], w.t.Name)
			if ticked[rec.ID] > 0 {
				words += ", " + doneWords(done, ticked[rec.ID])
			}
			out[rec.ID] = append(out[rec.ID], words)
		}
	}
	return out
}

// doneWords is how many are ticked, by the tick's own word: 1 done.
func doneWords(f *schema.Field, n int) string {
	return fmt.Sprintf("%d %s", n, f.Words())
}
