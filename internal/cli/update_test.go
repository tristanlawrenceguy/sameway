package cli_test

import (
	"strings"
	"testing"
)

// Keeping the program current is a command as well as a setting. A build
// from source says so instead of comparing itself to a release, which is
// the only path a test can take without reaching the network.
func TestUpdateSaysWhatABuildFromSourceCanDo(t *testing.T) {
	r := run(t, "", "update", "--check")
	if r.code != 1 || !strings.Contains(r.stderr, "rather than a version") || !strings.Contains(r.stderr, "make build") {
		t.Errorf("expected a clear refusal that says how to get a versioned build, got %+v", r)
	}
	if r := run(t, "", "--json", "update"); r.code != 1 || !strings.Contains(r.stderr, `"error"`) {
		t.Errorf("the refusal is JSON with --json: %+v", r)
	}
	if r := run(t, "", "help"); !strings.Contains(r.stdout, "sameway update") {
		t.Error("update should be in the usage")
	}
}
