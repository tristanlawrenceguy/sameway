package schema

// A record's day and whether it is done are asked in many places: its
// row's group, its calendar day, what it is related to by day, its export,
// what an import's date column fills, what a repeat moves. Each once
// decided for itself, and they disagreed: one took any datetime, hidden
// ones too, one a shown one, one starts first; one took done from a field
// named done and nothing else. These are the one answer each asks.

// DayField is the field a record's day comes from: starts when the type
// has a shown datetime field of that name, else its first shown datetime
// field, else "". Shown only, because a hidden datetime is the system's
// bookkeeping, not a day a person keeps; starts first, because an event's
// day is when it starts, wherever its fields are listed.
func (t *Type) DayField() string {
	if f, ok := t.Field("starts"); ok && f.Type == "datetime" && !f.Hidden {
		return f.Name
	}
	for _, f := range t.Fields {
		if f.Type == "datetime" && !f.Hidden {
			return f.Name
		}
	}
	return ""
}

// HasDay says whether a type's records have a day of their own, such as a
// task's due or an event's starts.
func (t *Type) HasDay() bool { return t.DayField() != "" }

// DoneField is the yes-or-no a thing is finished by, such as a task's done,
// or nil: a bool named done, completed, complete or finished. Pinned or
// archived is a setting, not something finished, so it is not one.
func (t *Type) DoneField() *Field {
	for i := range t.Fields {
		if f := &t.Fields[i]; f.Type == "bool" && isDoneName(f.Name) {
			return f
		}
	}
	return nil
}

func isDoneName(name string) bool {
	return name == "done" || name == "completed" || name == "complete" || name == "finished"
}

// Done says whether a record is finished: its done tick is on, or, for a
// type finished by a pick-list (a reminder's state), it is at done.
func (t *Type) Done(fields map[string]any) bool {
	name, done, _, ok := t.finishing()
	return ok && fields[name] == done
}
