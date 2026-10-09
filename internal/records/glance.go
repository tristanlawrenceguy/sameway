package records

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What a record says at a glance is worked out once, here, from its
// schema, and every surface shows the same facts its own way: the line
// under a record's title as chips, a list's row as short words, a block's
// list as plain text, the API's and the assistant's view as a line.
// Three functions once said it three ways, each patched by the type it
// was caught on, and the same record read "Due Fri 9 Oct 2026, 14:00" on
// the canvas and "In 4 days at 2:00pm" on its list. It is worked out from
// the schema and the store alone, so no surface needs the pages to say it.

// Fact is one thing a record says at a glance.
type Fact struct {
	Kind   string // done, state, flag, day, person, ref, count
	Field  string // the field it says, which the record's page need not list again
	Text   string // as a person reads it
	Tone   string // the badge's tone
	Person string // a person's id, for their colour
	Label  string // a person's field's label: For
	Short  string // a day as a row says it, a class to go with it
	Class  string
	When   string // a day's value, for its <time datetime>
	Full   string // a day in full, when its words leave the date out
}

// Glance is what a record says at a glance, by the schema's rules, not by
// its type's name:
//   - its done tick, when ticked;
//   - its stage when it says more than the tick (Doing), or its status or
//     state (Draft, Pending); a kind, a method, a cadence is the record's
//     own page's to say;
//   - a setting that is on, by its label (Pinned, On the canvas);
//   - its first day, in words, named by its field (Due Fri 9 Oct,
//     Starts tomorrow at 2pm), and "Overdue" in words and amber when it
//     has passed on a record that can be done and is not; a day that has
//     passed on anything else (a meeting, an entry) is only past;
//   - what it belongs to or who it is for, unless that is its title; a
//     file it points at is an attachment, not what it belongs to;
//   - what is in it: how many records name it as theirs, and how many of
//     them are done (3 tasks, 1 done), from in, what a listing counted
//     for all its rows at once, or counted for this one when in is nil
//     (glance_count.go).
//
// A day is said to the reader: as it is for them at now, on the 24-hour
// clock when h24.
//
// The order is always this one, so the same fact sits in the same place
// on every row (design/foundations/glance.md).
func Glance(st *store.Store, t *schema.Type, rec *store.Record, now time.Time, h24 bool, in Counts) []Fact {
	var out []Fact
	done := false
	if f := t.DoneField(); f != nil {
		if on, _ := rec.Fields[f.Name].(bool); on {
			done = true
			out = append(out, Fact{Kind: "done", Field: f.Name, Text: f.Display(), Tone: "success"})
		}
	}
	for _, f := range t.Shown() {
		v, _ := rec.Fields[f.Name].(string)
		if f.Type != "enum" || v == "" {
			continue
		}
		if worth, said := StateWorth(t, f, v); said && worth {
			out = append(out, Fact{Kind: "state", Field: f.Name, Text: f.ValueLabel(v), Tone: "info"})
		}
	}
	for _, f := range t.Shown() {
		if f.Type == "bool" && (t.DoneField() == nil || f.Name != t.DoneField().Name) {
			if on, _ := rec.Fields[f.Name].(bool); on {
				out = append(out, Fact{Kind: "flag", Field: f.Name, Text: f.Display(), Tone: "neutral"})
			}
		}
	}
	if d, ok := dayGlance(t, rec, done, now, h24); ok {
		out = append(out, d)
	}
	if f, ok := belongsTo(st, t, rec); ok {
		out = append(out, f)
	}
	if in == nil {
		in = CountsOf(st, t, []*store.Record{rec})
	}
	for _, words := range in[rec.ID] {
		out = append(out, Fact{Kind: "count", Text: words, Tone: "neutral"})
	}
	return out
}

// belongsTo is what a record belongs to or who it is for, by its first
// ref that names something other than a file, unless that is its title.
func belongsTo(st *store.Store, t *schema.Type, rec *store.Record) (Fact, bool) {
	title := strings.TrimSpace(Name(st, t, rec))
	for _, f := range t.Shown() {
		id, _ := rec.Fields[f.Name].(string)
		if f.Type != "ref" || id == "" || f.To == FileType {
			continue
		}
		name := RefTitle(st, f, id)
		if name == "" || name == title {
			return Fact{}, false
		}
		if f.To == PersonType {
			return Fact{Kind: "person", Field: f.Name, Text: f.Display() + " " + name, Person: id, Label: f.Display()}, true
		}
		return Fact{Kind: "ref", Field: f.Name, Text: name, Tone: "neutral"}, true
	}
	return Fact{}, false
}

// GlanceText is the facts as plain words, for a block's list, for the
// API's and the assistant's view of a record, and for an agent; in is as
// Glance takes it.
func GlanceText(st *store.Store, t *schema.Type, rec *store.Record, now time.Time, h24 bool, in Counts) string {
	var words []string
	for _, f := range Glance(st, t, rec, now, h24, in) {
		words = append(words, f.Text)
	}
	return strings.Join(words, " · ")
}

// RefTitle is what a ref names, by the record's name: the id when its
// type is not known, and that it is gone when the record is.
func RefTitle(st *store.Store, f schema.Field, id string) string {
	if id == "" {
		return ""
	}
	t, ok := st.Types().Get(f.To)
	if !ok {
		return id
	}
	rec, err := st.Get(f.To, id)
	if err != nil {
		return "a " + f.To + " that is no longer here"
	}
	return Name(st, t, rec)
}

// StateWorth says whether a field is a record's state (its stage, or a
// status or state) and whether its value is worth saying: a stage when it
// says more than the tick (Doing; To do and Done are the tick), a status
// or state when it is not where every record rests (Published, not
// Draft), or when it waits on someone (Pending).
func StateWorth(t *schema.Type, f schema.Field, v string) (worth, state bool) {
	if f.Type != "enum" {
		return false, false
	}
	if f.Name == t.Stage() {
		return t.SaysMoreThanTick(v), true
	}
	if f.Name != "status" && f.Name != "state" {
		return false, false
	}
	rest, _ := f.Default.(string)
	if rest == "" && len(f.Values) > 0 {
		rest = f.Values[0]
	}
	return v != rest || v == "pending", true
}
