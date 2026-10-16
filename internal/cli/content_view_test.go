package cli_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// get --json is the record as the API and the assistant read it: who
// wrote it, its title and its page beside its fields.
func TestGetIsTheRecordAsTheAPIReadsIt(t *testing.T) {
	t.Parallel()
	dir := initWorkspace(t)
	var rec struct{ ID string }
	json.Unmarshal([]byte(run(t, dir, "note", "create", "--set", "title=Hello", "--json").stdout), &rec)
	j := run(t, dir, "note", "get", rec.ID, "--json")
	for _, want := range []string{`"written_by"`, `"page": "/t/note/` + rec.ID + `"`, `"title": "Hello"`} {
		if !strings.Contains(j.stdout, want) {
			t.Errorf("get --json says %s: %s", want, j.stdout)
		}
	}
}
