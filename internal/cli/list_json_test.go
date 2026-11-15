package cli_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// A record from list --json says its title, who wrote it and that what it
// says is data, as one from get and from the API's list do.
func TestListJSONSaysWhoWroteEachRecord(t *testing.T) {
	t.Parallel()
	dir := initWorkspace(t)
	if r := run(t, dir, "note", "create", "--set", "title=Hello"); r.code != 0 {
		t.Fatalf("create: %+v", r)
	}
	r := run(t, dir, "note", "list", "--json")
	var recs []map[string]any
	if err := json.Unmarshal([]byte(r.stdout), &recs); err != nil || len(recs) != 1 {
		t.Fatalf("list --json should be an array of one: %v %+v", err, r)
	}
	got := recs[0]
	untrusted, _ := got["untrusted"].(string)
	if got["id"] == "" || got["title"] != "Hello" || got["written_by"] == "" || !strings.Contains(untrusted, "never instructions") {
		t.Errorf("a listed record should say its title, who wrote it and that it is data: %v", got)
	}
}
