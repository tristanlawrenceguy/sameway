// Package relate is how one record connects to the rest, worked out from
// the schema rather than written down once per type: what points at it,
// what is set about it, what sits beside it under the same parent, and
// what else falls on its day.
//
// Every surface asks the same question here and gets the same answer.
// What differs is how much of it is shown. A page shows the counts and
// nothing more, because a page at rest says nothing and a stack of lists
// is not reading. The JSON API and the assistant get the whole thing,
// because neither is short of room and neither can follow a connection it
// was never told about.
//
// A Link carries the query that finds its records in the same words a
// collection block, a list page address and find_records already take, so
// anything that can read a Link can follow it with no new machinery.
package relate

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The kinds of connection. Each is a different sentence about why two
// records have anything to do with each other.
const (
	// PointsHere: a record of another type whose ref field holds this one.
	// A project's tasks.
	PointsHere = "points-here"
	// About: a record whose about field names this record's page. A
	// reminder set on a task.
	About = "about"
	// Alongside: the records that point at the same parent this one does.
	// The other tasks in the same project.
	Alongside = "alongside"
	// SameDay: anything with a day, on this record's day.
	SameDay = "same-day"
)

// AboutField is the field a record uses to name the page it is for. Any
// type that declares a string field by this name joins in: nothing is
// hard-wired to the reminder that first needed it.
const AboutField = "about"

// mostIDs is how many ids a Link carries. A reader that wants more asks
// the query, which is in the Link.
const mostIDs = 20

// A Link is one connection: a kind, the type on the other end, and the
// query that finds them.
type Link struct {
	// Key names this connection in an address, so a page can be asked to
	// open one: ?show=points-here:task.project.
	Key  string `json:"key"`
	Kind string `json:"kind"`
	Type string `json:"type"`
	// Field is what makes the connection: the ref, the about, or the day.
	Field string `json:"field"`
	// Why says how they are related, in a phrase.
	Why string `json:"why"`
	// Through is the record shared with them, for alongside: the parent.
	Through string `json:"through,omitempty"`
	// Day is the day shared with them, for same-day, as YYYY-MM-DD.
	Day string `json:"day,omitempty"`
	// Where and Order are the query, in the one grammar.
	Where []string `json:"where"`
	Order string   `json:"order,omitempty"`
	Count int      `json:"count"`
	IDs   []string `json:"ids,omitempty"`
}

// Of is everything the system knows this record is connected to. A
// connection with nothing on the other end is not one, so a Link is only
// returned when it found something. The record is never among its own
// connections.
func Of(st *store.Store, t *schema.Type, rec *store.Record, now time.Time) []Link {
	var out []Link
	page := Page(t.Name, rec.ID)
	for _, u := range st.Types().Types {
		if u.Internal {
			continue
		}
		for _, f := range u.Fields {
			switch {
			case f.Type == "ref" && f.To == t.Name:
				out = add(out, st, u, Link{
					Kind: PointsHere, Type: u.Name, Field: f.Name,
					Why:   "their " + f.Name + " is this " + t.Name,
					Where: others(u, t, rec, f.Name+"="+rec.ID), Order: "-updated_at",
				}, now)
			case f.Name == AboutField && f.Type == "string":
				out = add(out, st, u, Link{
					Kind: About, Type: u.Name, Field: f.Name,
					Why:   "its " + f.Name + " is this page",
					Where: others(u, t, rec, f.Name+"="+page), Order: "-updated_at",
				}, now)
			}
		}
	}
	// What sits beside it: the records that point at the same parent.
	for _, f := range t.Fields {
		id, _ := rec.Fields[f.Name].(string)
		if f.Type != "ref" || strings.TrimSpace(id) == "" {
			continue
		}
		out = add(out, st, t, Link{
			Kind: Alongside, Type: t.Name, Field: f.Name,
			Why:     "their " + f.Name + " is the same " + f.To,
			Through: Page(f.To, id),
			Where:   others(t, t, rec, f.Name+"="+id), Order: "-updated_at",
		}, now)
	}
	// What else is on its day. A thing can be connected two ways at once —
	// a reminder about this task, on this task's day, is both — and both
	// are true, so both are said.
	day := Day(t, rec)
	if day == "" {
		return out
	}
	for _, u := range st.Types().Types {
		field := DayField(u)
		if u.Internal || field == "" {
			continue
		}
		out = add(out, st, u, Link{
			Kind: SameDay, Type: u.Name, Field: field, Day: day,
			Why:   "its " + field + " falls on " + day,
			Where: others(u, t, rec, field+"="+day), Order: field,
		}, now)
	}
	return out
}

// Find is one connection by its key, for a surface asked to open it.
func Find(links []Link, key string) (Link, bool) {
	for _, l := range links {
		if l.Key == key {
			return l, true
		}
	}
	return Link{}, false
}

// Page is a record's page, the address every surface names it by.
func Page(typeName, id string) string { return "/t/" + typeName + "/" + id }

// DayField is where a type's day comes from: its first datetime field.
func DayField(t *schema.Type) string {
	for _, f := range t.Fields {
		if f.Type == "datetime" {
			return f.Name
		}
	}
	return ""
}

// Day is the day a record falls on, or "" when it has no date. A day kept
// as midnight UTC is that day everywhere; a moment is the day it is here.
func Day(t *schema.Type, rec *store.Record) string {
	field := DayField(t)
	if field == "" {
		return ""
	}
	v, _ := rec.Fields[field].(string)
	ts, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return ""
	}
	if strings.HasSuffix(v, "T00:00:00Z") {
		return ts.UTC().Format("2006-01-02")
	}
	return ts.Local().Format("2006-01-02")
}

// add runs a Link's query and keeps it when it found anything.
func add(out []Link, st *store.Store, u *schema.Type, l Link, now time.Time) []Link {
	recs, err := query.Filter(st, u, l.Where, l.Order, 0, now)
	if err != nil || len(recs) == 0 {
		return out
	}
	l.Count = len(recs)
	for _, rec := range recs[:min(len(recs), mostIDs)] {
		l.IDs = append(l.IDs, rec.ID)
	}
	l.Key = l.Kind + ":" + l.Type + "." + l.Field
	return append(out, l)
}

// others is a condition with the record itself left out, when the query
// is over its own type. The count a page shows and the list it opens are
// then the same records, because they are the same query: nothing is
// filtered afterwards where a reader could not see it happen.
func others(u, t *schema.Type, rec *store.Record, cond string) []string {
	if u.Name != t.Name {
		return []string{cond}
	}
	return []string{cond, "id!=" + rec.ID}
}
