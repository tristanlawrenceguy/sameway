package schema

import "testing"

func parsed(t *testing.T, src string) *Type {
	t.Helper()
	typ, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return typ
}

// A type's day is starts when it has one, else its first shown datetime;
// a hidden datetime is never it.
func TestDayFieldPrefersStartsAndSkipsHidden(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{"name: meeting\nfields:\n  title: {type: string}\n  ends: {type: datetime}\n  starts: {type: datetime}\n", "starts"},
		{"name: task\nfields:\n  title: {type: string}\n  seen: {type: datetime, hidden: true}\n  due: {type: datetime}\n", "due"},
		{"name: note\nfields:\n  title: {type: string}\n  seen: {type: datetime, hidden: true}\n", ""},
		{"name: meeting\nfields:\n  starts: {type: datetime, hidden: true}\n  at: {type: datetime}\n", "at"},
	} {
		typ := parsed(t, c.src)
		if got := typ.DayField(); got != c.want {
			t.Errorf("%s: DayField = %q, want %q", typ.Name, got, c.want)
		}
		if typ.HasDay() != (c.want != "") {
			t.Errorf("%s: HasDay = %v", typ.Name, typ.HasDay())
		}
	}
	// A repeat moves the same day.
	typ := parsed(t, "name: meeting\nfields:\n  ends: {type: datetime}\n  starts: {type: datetime}\n  repeat: {type: repeat}\n")
	if _, day, ok := typ.Repeats(); !ok || day != "starts" {
		t.Errorf("Repeats day = %q, want starts", day)
	}
}

// Done is a tick by any of its names, or a pick-list at done; pinned is not.
func TestDoneFieldAndDone(t *testing.T) {
	task := parsed(t, "name: task\nfields:\n  pinned: {type: bool}\n  completed: {type: bool}\n")
	if f := task.DoneField(); f == nil || f.Name != "completed" {
		t.Fatalf("DoneField = %v, want completed", f)
	}
	if !task.Done(map[string]any{"completed": true}) || task.Done(map[string]any{"pinned": true}) {
		t.Error("a task is done by its completed tick alone")
	}
	reminder := parsed(t, "name: reminder\nfields:\n  state: {type: enum, values: [waiting, ringing, done]}\n")
	if reminder.DoneField() != nil {
		t.Error("a reminder has no tick")
	}
	if !reminder.Done(map[string]any{"state": "done"}) || reminder.Done(map[string]any{"state": "ringing"}) {
		t.Error("a reminder is done when its state is done")
	}
	note := parsed(t, "name: note\nfields:\n  pinned: {type: bool}\n")
	if note.DoneField() != nil || note.Done(map[string]any{"pinned": true}) {
		t.Error("pinned is a setting, not done")
	}
}

// A field is named by its label, else its name with spaces; in a sentence
// it is lower case, an acronym kept.
func TestFieldDisplayAndWords(t *testing.T) {
	typ := parsed(t, "name: link\nfields:\n  follow_up: {type: datetime}\n  by: {type: datetime, label: Goal by}\n  url: {type: string, label: URL}\n")
	for _, c := range []struct{ name, display, words string }{
		{"follow_up", "Follow up", "follow up"},
		{"by", "Goal by", "goal by"},
		{"url", "URL", "URL"},
		{"not_here", "Not here", "not here"},
	} {
		if got := typ.FieldDisplay(c.name); got != c.display {
			t.Errorf("FieldDisplay(%s) = %q, want %q", c.name, got, c.display)
		}
		if got := typ.FieldWords(c.name); got != c.words {
			t.Errorf("FieldWords(%s) = %q, want %q", c.name, got, c.words)
		}
	}
}

// A record is called by its title, else the first thing it says, else its
// kind and id.
func TestCalled(t *testing.T) {
	typ := parsed(t, "name: call_log\ntitle: subject\nfields:\n  subject: {type: string}\n  kind: {type: enum, values: [in_person, phone]}\n")
	for _, c := range []struct {
		fields map[string]any
		want   string
	}{
		{map[string]any{"subject": "Rent", "kind": "phone"}, "Rent"},
		{map[string]any{"subject": " ", "kind": "in_person"}, "In person"},
		{map[string]any{}, "call log abc"},
	} {
		if got := typ.Called("abc", c.fields); got != c.want {
			t.Errorf("Called(%v) = %q, want %q", c.fields, got, c.want)
		}
	}
}
