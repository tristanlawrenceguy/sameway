package schema

import "testing"

// TestPlural: every place a type's name is made plural says it the same
// way, and says it right ("activitys" was filed four times: backlog 0052,
// 0078, 0093, 0104).
func TestPlural(t *testing.T) {
	for in, want := range map[string]string{
		"activity": "activities", "category": "categories", "entry": "entries",
		"day": "days", "key": "keys", "note": "notes", "task": "tasks",
		"box": "boxes", "match": "matches", "wish": "wishes",
		"person": "people", "child": "children", "mouse": "mice",
		"news": "news", "series": "series", "": "",
	} {
		if got := Plural(in); got != want {
			t.Errorf("Plural(%q) = %q, want %q", in, got, want)
		}
	}
}
