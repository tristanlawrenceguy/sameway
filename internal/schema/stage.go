package schema

// A task says two things about being finished: its done tick, which a
// list, a calendar and a repeat go by, and its status (To do, Doing,
// Done), which a board goes by. They are one fact, so they never
// disagree: every write keeps them in step here, and a record from before
// the status existed reads its status from its tick.
//
// The rule applies to any type with both a finishing tick (a bool named
// done, as finishing says) and a pick-list with a done choice that is not
// its first: the pick-list is the type's stage.

// stage names a type's tick and its stage pick-list; ok is false for a
// type without both.
func (t *Type) stage() (flag string, stage *Field, ok bool) {
	var f *Field
	for i := range t.Fields {
		g := &t.Fields[i]
		switch {
		case g.Type == "bool" && flag == "" && isDoneName(g.Name):
			flag = g.Name
		case g.Type == "enum" && f == nil && contains(g.Values, "done") && g.Values[0] != "done":
			f = g
		}
	}
	return flag, f, flag != "" && f != nil
}

// Stage names the pick-list a type keeps in step with its done tick, such
// as a task's status; "" for a type that has none.
func (t *Type) Stage() string {
	if _, f, ok := t.stage(); ok {
		return f.Name
	}
	return ""
}

// SaysMoreThanTick says whether a stage value tells a person something
// the tick does not: Doing does; Done and To do are the tick itself, so a
// task's row need not say them twice.
func (t *Type) SaysMoreThanTick(value string) bool {
	_, f, ok := t.stage()
	return ok && value != "done" && value != notDone(f)
}

// notDone is where a stage goes when its tick is taken off: its default,
// else its first choice (To do).
func notDone(f *Field) string {
	if d, ok := f.Default.(string); ok && d != "done" && contains(f.Values, d) {
		return d
	}
	return f.Values[0]
}

// unstored gives a record the half of the pair nothing was said for,
// from the half that was: a status from the tick (Done when ticked, else
// To do), or a tick from the status. It is what a record written before
// the status existed reads as, and what a new record gets when only one
// was given. unset names the fields that had nothing; they already hold
// their defaults.
func (t *Type) unstored(fields map[string]any, unset func(name string) bool) {
	flag, f, ok := t.stage()
	if !ok {
		return
	}
	switch {
	case unset(f.Name) && !unset(flag):
		if fields[flag] == true {
			fields[f.Name] = "done"
		} else {
			fields[f.Name] = notDone(f)
		}
	case unset(flag) && !unset(f.Name):
		fields[flag] = fields[f.Name] == "done"
	}
}

// Unstored is unstored for a record read back: missing names the fields
// nothing was stored for.
func (t *Type) Unstored(fields map[string]any, missing map[string]bool) {
	t.unstored(fields, func(name string) bool { return missing[name] })
}

// KeepInStep makes a record about to be stored say one thing about being
// finished. before is what it was, or nil for a record written whole.
// Only the status changed (a board's Move): the tick follows, ticked for
// Done and not for anything else. Otherwise the status follows the tick:
// ticked is Done; unticked, a Done task goes back to To do, and a task
// that is To do or Doing stays where it is. That covers a tick alone, a
// contradiction sent in one change, and a record written whole, where
// the tick is what every list and calendar has always shown.
//
// Unticking does not remember Doing: a task ticked from Doing and then
// unticked is To do, since the tick is all an untick says. Undo puts back
// the whole record, so undoing the tick does bring back Doing.
func (t *Type) KeepInStep(before, clean map[string]any) {
	flag, f, ok := t.stage()
	if !ok {
		return
	}
	done := clean[flag] == true
	at, _ := clean[f.Name].(string)
	if (at == "done") == done {
		return
	}
	if before != nil && before[f.Name] != clean[f.Name] && before[flag] == clean[flag] {
		clean[flag] = at == "done"
		return
	}
	if done {
		clean[f.Name] = "done"
	} else {
		clean[f.Name] = notDone(f)
	}
}
