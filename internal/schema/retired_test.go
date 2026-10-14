package schema

import "testing"

// A reminder's state named "Ringing" in a copy of the type made before is
// brought up to "Rang", as Sameway now names it; a name the person gave
// the state is theirs and is kept.
func TestARetiredNameIsBroughtUpToDate(t *testing.T) {
	t.Parallel()
	builtin := &Set{byName: map[string]*Type{}}
	b := &Type{Name: "reminder", Provided: true, Fields: []Field{{Name: "state", Type: "enum", Values: []string{"set", "rang", "done"}, Labels: map[string]string{"set": "Waiting", "rang": "Rang", "done": "Done"}}}}
	builtin.Types, builtin.byName["reminder"] = []*Type{b}, b
	for old, want := range map[string]string{"Ringing": "Rang", "Went off": "Went off"} {
		ws := &Set{byName: map[string]*Type{}}
		copy := &Type{Name: "reminder", Fields: []Field{{Name: "state", Type: "enum", Values: []string{"set", "rang", "done"}, Labels: map[string]string{"set": "Waiting", "rang": old, "done": "Done"}}}}
		ws.Types, ws.byName["reminder"] = []*Type{copy}, copy
		ws.Complete(builtin)
		if f, _ := copy.Field("state"); f.ValueLabel("rang") != want {
			t.Errorf("%q should become %q, got %q", old, want, f.ValueLabel("rang"))
		}
	}
}
