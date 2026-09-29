package cli_test

import (
	"encoding/json"
	"testing"
)

// TestListJSONEmptyProducesArray covers the bug where `sameway <type> list --json`
// with zero records produced JSON null instead of an empty array [].
func TestListJSONEmptyProducesArray(t *testing.T) {
	dir := initWorkspace(t)

	r := run(t, dir, "note", "list", "--json")
	if r.code != 0 {
		t.Fatalf("list --json with no notes: exit code %d stderr=%q", r.code, r.stderr)
	}

	var data any
	if err := json.Unmarshal([]byte(r.stdout), &data); err != nil {
		t.Fatalf("stdout is not valid JSON: %s\nerror: %v", r.stdout, err)
	}

	arr, ok := data.([]any)
	if !ok {
		t.Errorf("expected a JSON array, got %T (%q)", data, r.stdout)
		return
	}
	if len(arr) != 0 {
		t.Errorf("expected zero records, got %d", len(arr))
	}

	// Also check the raw output is literally "[]" (possibly with whitespace).
	raw := []byte(r.stdout)
	for len(raw) > 0 && (raw[0] == ' ' || raw[0] == '\n' || raw[0] == '\t') {
		raw = raw[1:]
	}
	if len(raw) > 0 && raw[len(raw)-1] == '\n' {
		raw = raw[:len(raw)-1]
	}
	if string(raw) != "[]" {
		t.Errorf("expected \"[]\", got %q", r.stdout)
	}
}

// TestListJSONEmptyAfterDelete covers the case where records existed but were
// all deleted — the list --json must still return [].
func TestListJSONEmptyAfterDelete(t *testing.T) {
	dir := initWorkspace(t)

	// Create a record.
	create := run(t, dir, "note", "create", "--set", "title=Hello", "--json")
	if create.code != 0 {
		t.Fatalf("create: %+v", create)
	}

	var rec struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(create.stdout), &rec); err != nil || rec.ID == "" {
		// try a different unmarshal approach — the ID field might be nested differently
		var raw map[string]any
		if err := json.Unmarshal([]byte(create.stdout), &raw); err != nil {
			t.Fatalf("create output parse: %s\nerror: %v", create.stdout, err)
		}
		rec.ID = raw["id"].(string)
	}

	// Delete it.
	del := run(t, dir, "note", "delete", rec.ID)
	if del.code != 0 {
		t.Fatalf("delete: %+v", del)
	}

	// List should be empty array.
	r := run(t, dir, "note", "list", "--json")
	if r.code != 0 {
		t.Fatalf("list --json after delete: exit code %d stderr=%q", r.code, r.stderr)
	}

	var data any
	if err := json.Unmarshal([]byte(r.stdout), &data); err != nil {
		t.Fatalf("stdout is not valid JSON: %s\nerror: %v", r.stdout, err)
	}
	arr, ok := data.([]any)
	if !ok {
		t.Errorf("expected a JSON array after delete, got %T (%q)", data, r.stdout)
		return
	}
	if len(arr) != 0 {
		t.Errorf("expected zero records after delete, got %d", len(arr))
	}
}
