package schema

import "testing"

// TestValueLabel: a value reads as the schema names it, else as itself
// made readable; an acronym stays as it is.
func TestValueLabel(t *testing.T) {
	f := Field{Type: "enum", Values: []string{"reach", "in_progress", "GET"}, Labels: map[string]string{"reach": "At least the target"}}
	for v, want := range map[string]string{"reach": "At least the target", "in_progress": "In progress", "GET": "GET", "": ""} {
		if got := f.ValueLabel(v); got != want {
			t.Errorf("ValueLabel(%q) = %q, want %q", v, got, want)
		}
	}
}

// TestCompleteBringsValueLabels: a workspace made before its enum values
// had names gets the built-in names, for the values its copy has; a
// workspace that named them itself keeps its own.
func TestCompleteBringsValueLabels(t *testing.T) {
	ws := &Set{byName: map[string]*Type{}}
	old := &Type{Name: "habit", Fields: []Field{{Name: "aim", Type: "enum", Values: []string{"reach", "limit"}}, {Name: "mine", Type: "enum", Values: []string{"a"}, Labels: map[string]string{"a": "Mine"}}}}
	ws.Types, ws.byName["habit"] = []*Type{old}, old
	builtin := &Set{Types: []*Type{{Name: "habit", Provided: true, Fields: []Field{
		{Name: "aim", Type: "enum", Values: []string{"reach", "limit", "record"}, Labels: map[string]string{"reach": "At least the target", "record": "Just keep a record"}},
		{Name: "mine", Type: "enum", Values: []string{"a"}, Labels: map[string]string{"a": "Theirs"}},
	}}}}
	ws.Complete(builtin)
	aim, _ := old.Field("aim")
	if aim.ValueLabel("reach") != "At least the target" || aim.ValueLabel("limit") != "Limit" || aim.Labels["record"] != "" {
		t.Errorf("the workspace's aim should take the names for its own values: %v", aim.Labels)
	}
	if mine, _ := old.Field("mine"); mine.ValueLabel("a") != "Mine" {
		t.Errorf("a workspace's own names win: %v", mine.Labels)
	}
}
