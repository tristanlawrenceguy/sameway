package cli_test

import (
	"strings"
	"testing"
)

// A list goes out from the command line in any of its formats, and a
// format that does not fit says which do.
func TestAListGoesOutFromTheCommandLine(t *testing.T) {
	ws := initWorkspace(t)
	run(t, ws, "task", "create", "--set", "title=Order compost", "--set", "due=2026-10-05")
	r := run(t, ws, "task", "list", "--format", "csv")
	if r.code != 0 || !strings.Contains(r.stdout, "Title,") || !strings.Contains(r.stdout, "Order compost") {
		t.Fatalf("a CSV of the tasks: %d %q %q", r.code, r.stdout, r.stderr)
	}
	if r := run(t, ws, "task", "list", "--format", "ics"); r.code != 0 || !strings.Contains(r.stdout, "SUMMARY:Order compost") {
		t.Errorf("a calendar of the tasks: %q %q", r.stdout, r.stderr)
	}
	if r := run(t, ws, "task", "list", "--format", "vcf"); r.code == 0 || !strings.Contains(r.stderr, "it can be csv, xlsx, ics") {
		t.Errorf("tasks are not contacts, and it says what they can be: %q", r.stderr)
	}
}
