package cli_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// A command's --help says its lines on stdout and does nothing else:
// chat --help asked the model, restore --help restored a copy named -help.
func TestCommandHelpSaysItsLines(t *testing.T) {
	t.Parallel()
	dir := initWorkspace(t)
	for _, cmd := range []string{"init", "open", "serve", "describe", "check", "update", "search", "add", "agent", "look", "export", "import", "snapshots", "restore", "chat", "mcp", "connect", "discover", "component"} {
		for _, args := range [][]string{{cmd, "--help"}, {cmd, "-h"}, {"help", cmd}} {
			r := run(t, dir, args...)
			if r.code != 0 || !strings.Contains(r.stdout, "sameway "+cmd) || r.stderr != "" {
				t.Errorf("%v: %+v", args, r)
			}
		}
	}
	if r := run(t, dir, "note", "list", "--help"); r.code != 0 || !strings.Contains(r.stdout, "sameway <type> list") {
		t.Errorf("a content type's --help: %+v", r)
	}
	if r := run(t, dir, "help"); !strings.Contains(r.stdout, "Exit status: 0 done, 1 it went wrong, 2 the command was not understood") {
		t.Errorf("help says the exit codes: %s", r.stdout)
	}
}

// A typo is answered with what it most likely meant, as git does, and exits
// 2, a command line not understood; nothing is run.
func TestTyposSayWhatWasMeant(t *testing.T) {
	t.Parallel()
	dir := initWorkspace(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"descrbe"}, "Did you mean sameway describe?"},
		{[]string{"note", "lst"}, "Did you mean sameway note list?"},
		{[]string{"nots", "list"}, "Did you mean sameway note?"},
		{[]string{"help", "serach"}, "Did you mean sameway help search?"},
		// A flag alone is not a double-click: it opened the workspace.
		{[]string{"-x"}, "sameway has no flag -x"},
	} {
		r := run(t, dir, c.args...)
		if r.code != 2 || !strings.Contains(r.stderr, c.want) || r.stdout != "" {
			t.Errorf("%v: %+v (want %q)", c.args, r, c.want)
		}
	}
}

// A flag the command does not have is said once, with its flags, and
// exits 2; with --json it is the one JSON error.
func TestUnknownFlagSaidOnce(t *testing.T) {
	t.Parallel()
	dir := initWorkspace(t)
	r := run(t, dir, "serve", "--bogus")
	if r.code != 2 || strings.Count(r.stderr, "not defined") != 1 || !strings.Contains(r.stderr, "-addr") {
		t.Errorf("serve --bogus: %+v", r)
	}
	r = run(t, dir, "--json", "note", "lst")
	var e struct{ Error string }
	if r.code != 2 || json.Unmarshal([]byte(r.stderr), &e) != nil || !strings.Contains(e.Error, "list") {
		t.Errorf("--json usage error: %+v", r)
	}
}

// A record not there names what was asked for and how to see those there are.
func TestMissingRecordSaysHowToFindOne(t *testing.T) {
	t.Parallel()
	dir := initWorkspace(t)
	for _, verb := range []string{"get", "delete"} {
		r := run(t, dir, "note", verb, "nope")
		if r.code != 1 || !strings.Contains(r.stderr, `no note has the id "nope"`) || !strings.Contains(r.stderr, "sameway note list") {
			t.Errorf("%s: %+v", verb, r)
		}
	}
}

// An empty list is [] to an agent, never null.
func TestEmptyListsAreArrays(t *testing.T) {
	t.Parallel()
	dir := initWorkspace(t)
	for _, args := range [][]string{{"agent", "list", "--json"}, {"snapshots", "--json"}} {
		if r := run(t, dir, args...); r.code != 0 || strings.TrimSpace(r.stdout) != "[]" {
			t.Errorf("%v: %+v", args, r)
		}
	}
	if r := run(t, dir, "search", "zzzq", "--json"); !strings.Contains(r.stdout, `"hits": []`) {
		t.Errorf("search hits: %s", r.stdout)
	}
}
