package schema

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A thing that repeats is one record, not one per time: a task that is
// done every Tuesday is the same task, due again. When it is finished, by
// a tick, a dismissal, the assistant, the API or the command line, it
// comes back unfinished on its next day, in the same change, so the log
// has one entry and one Undo puts back the day it was and the tick.

// Repeats names a type's repeat field and the day it moves: its first
// repeat field and its first datetime field. ok is false for a type that
// has not both.
func (t *Type) Repeats() (repeat, day string, ok bool) {
	for _, f := range t.Fields {
		if f.Type == "repeat" && repeat == "" {
			repeat = f.Name
		}
		if f.Type == "datetime" && day == "" {
			day = f.Name
		}
	}
	return repeat, day, repeat != "" && day != ""
}

// finishing is the field a record is finished by, the value that says so
// and the value it goes back to: a task's done, true and false; a
// reminder's state, done and its first value.
func (t *Type) finishing() (name string, done, again any, ok bool) {
	for _, f := range t.Fields {
		switch {
		case f.Type == "bool" && (f.Name == "done" || f.Name == "completed" || f.Name == "complete" || f.Name == "finished"):
			return f.Name, true, false, true
		case f.Type == "enum" && contains(f.Values, "done") && f.Values[0] != "done":
			again := any(f.Values[0])
			if f.Default != nil {
				again = f.Default
			}
			return f.Name, "done", again, true
		}
	}
	return "", nil, nil, false
}

// Advance moves a repeating record that has just been finished to its
// next time: clean is the record about to be stored, before what it was.
// It says whether it moved; a repeat that has ended leaves it finished.
// A monthly or yearly repeat that named no day keeps the day it had, so
// the 31st stays the 31st after February (when.Pin).
func (t *Type) Advance(before, clean map[string]any, now time.Time) bool {
	repeat, day, ok := t.Repeats()
	if !ok {
		return false
	}
	name, done, again, ok := t.finishing()
	rule, _ := clean[repeat].(string)
	if !ok || rule == "" || clean[name] != done || before[name] == done {
		return false
	}
	v, _ := clean[day].(string)
	rule = when.Pin(rule, v)
	next, ok := when.Next(rule, v, now)
	if !ok {
		return false
	}
	clean[name], clean[day], clean[repeat] = again, next, when.DropTime(rule)
	// Due again, a task is To do again, not Done (stage.go).
	if flag, f, ok := t.stage(); ok && flag == name {
		clean[f.Name] = notDone(f)
	}
	return true
}

// Advanced says whether a change that was sent to finish a repeating
// record moved it on instead: sent finished it, and what was saved is
// not finished and still repeats. It is how what was done is said.
func (t *Type) Advanced(sent, saved map[string]any) bool {
	repeat, _, ok := t.Repeats()
	name, done, _, finishes := t.finishing()
	if !ok || !finishes || saved[repeat] == "" || saved[name] == done {
		return false
	}
	// A board's Move to Done finishes a task as its tick does (stage.go).
	if flag, g, ok := t.stage(); ok && flag == name && sent[g.Name] == "done" {
		return true
	}
	if sent[name] == nil {
		return false
	}
	f, _ := t.Field(name)
	v, err := coerce(*f, sent[name])
	return err == nil && v == done
}
