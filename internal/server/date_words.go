package server

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The words that tell two records alike apart, said the way every other
// day is said (internal/when): this file once had its own, which said a
// moment in UTC after the glance had stopped doing so.

// dayWords is the record's first day as a few words after its title, the
// field named: due tomorrow, with the year when it is not this one.
func dayWords(t *schema.Type, rec *store.Record) string {
	for _, kind := range []string{"datetime", "time", "date"} {
		for _, f := range t.Shown() {
			if f.Type != kind {
				continue
			}
			v := str(rec.Fields[f.Name], f.Name)
			if v == "" || v == f.Name {
				continue
			}
			if kind == "date" {
				return "on " + whenWords(v)
			}
			return "due " + whenWords(v)
		}
	}
	return ""
}

// whenWords is a stored day or moment as a person plans by it.
func whenWords(v string) string {
	if _, err := time.Parse(time.RFC3339, v); err != nil {
		return ""
	}
	return when.Relative(v, time.Now())
}

// momentWords is a moment as a person plans by it: Today at 3pm.
func momentWords(at time.Time) string { return when.At(at, time.Now()) }

// shortDay is a date as a person plans by it: Today, Fri 9 Oct.
func shortDay(d time.Time) string { return when.Day(d, time.Now()) }
