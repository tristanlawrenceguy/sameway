package blocks

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The words that tell two records alike apart, said the way every other
// day is said (internal/when), to the person reading (reader.go): this
// file once had its own, which said a moment in UTC after the glance had
// stopped doing so.

// dayWords is the record's day (schema DayField) as a few words after its
// title: due tomorrow, with the year when it is not this one.
func (w *Workspace) dayWords(t *schema.Type, rec *store.Record) string {
	day := t.DayField()
	v := str(rec.Fields[day], day)
	if day == "" || v == "" || v == day {
		return ""
	}
	return "due " + w.whenWords(v)
}

// whenWords is a stored day or moment as a person plans by it.
func (w *Workspace) whenWords(v string) string {
	if _, err := time.Parse(time.RFC3339, v); err != nil {
		return ""
	}
	return when.Relative(v, w.now(), w.H24())
}

// MomentWords is a moment as a person plans by it: Today at 3pm.
func (w *Workspace) MomentWords(at time.Time) string { return when.At(at, w.now(), w.H24()) }

// shortDay is a date as a person plans by it: Today, Fri 9 Oct.
func (w *Workspace) shortDay(d time.Time) string { return when.Day(d, w.now()) }
