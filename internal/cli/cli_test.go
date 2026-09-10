package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/cli"
)

type result struct {
	code   int
	stdout string
	stderr string
}

// run drives the real command with a workspace dir, the way a terminal agent would.
func run(t *testing.T, dir string, args ...string) result {
	t.Helper()
	var out, errOut bytes.Buffer
	code := cli.Run(args, cli.Env{Stdout: &out, Stderr: &errOut, Dir: dir})
	return result{code, out.String(), errOut.String()}
}

func initWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	r := run(t, dir, "init", dir, "--no-detect")
	if r.code != 0 {
		t.Fatalf("init failed: %s", r.stderr)
	}
	// Tests must never dial a model.
	os.WriteFile(filepath.Join(dir, "workspace.yaml"), []byte("name: Test\nllm:\n  provider: none\n"), 0o644)
	return dir
}

func TestHelpAndVersion(t *testing.T) {
	if r := run(t, "", "help"); r.code != 0 || !strings.Contains(r.stdout, "Usage:") {
		t.Errorf("help: %+v", r)
	}
	if r := run(t, ""); r.code != 0 || !strings.Contains(r.stdout, "Usage:") {
		t.Errorf("no args should print usage: %+v", r)
	}
	if r := run(t, "", "--version"); r.code != 0 || !strings.HasPrefix(r.stdout, "sameway ") {
		t.Errorf("version: %+v", r)
	}
}

func TestNoWorkspaceIsAClearError(t *testing.T) {
	r := run(t, t.TempDir(), "describe")
	if r.code != 1 || !strings.Contains(r.stderr, "sameway init") {
		t.Errorf("expected a hint to run init, got %+v", r)
	}
}

func TestInitRefusesToOverwrite(t *testing.T) {
	dir := initWorkspace(t)
	if r := run(t, dir, "init", dir, "--no-detect"); r.code == 0 || !strings.Contains(r.stderr, "--force") {
		t.Errorf("second init should fail and mention --force: %+v", r)
	}
	if r := run(t, dir, "init", dir, "--force", "--no-detect"); r.code != 0 || !strings.Contains(r.stdout, "Edit ") || strings.Contains(r.stdout, "No local model server") {
		t.Errorf("init --force --no-detect: %+v", r)
	}
}

// TestDescribeJSON is what an agent reads first.
func TestDescribeJSON(t *testing.T) {
	dir := initWorkspace(t)
	r := run(t, dir, "describe", "--json")
	if r.code != 0 {
		t.Fatalf("%+v", r)
	}
	var d struct {
		Types      []struct{ Name string }
		Components []struct{ Name, Source string }
		LLM        struct {
			Ready   bool
			Problem string
		}
	}
	if err := json.Unmarshal([]byte(r.stdout), &d); err != nil {
		t.Fatalf("describe --json is not JSON: %v\n%s", err, r.stdout)
	}
	if len(d.Types) != 5 || len(d.Components) < 19 || d.LLM.Ready || d.LLM.Problem == "" {
		t.Errorf("describe content: %+v", d)
	}
	human := run(t, dir, "describe")
	if !strings.Contains(human.stdout, "Content types:") || !strings.Contains(human.stdout, "note") {
		t.Errorf("human describe: %s", human.stdout)
	}
}

// TestContentCommands covers create, list, get, update, delete in both
// human and JSON modes, with flags in the positions people actually type.
func TestContentCommands(t *testing.T) {
	dir := initWorkspace(t)

	created := run(t, dir, "note", "create", "--set", "title=Hello", "--set", "tags=a, b", "--json")
	if created.code != 0 {
		t.Fatalf("create: %+v", created)
	}
	var rec struct {
		ID     string
		Fields map[string]any
	}
	json.Unmarshal([]byte(created.stdout), &rec)
	if rec.ID == "" || rec.Fields["title"] != "Hello" {
		t.Fatalf("create output: %s", created.stdout)
	}
	if tags, _ := rec.Fields["tags"].([]any); len(tags) != 2 {
		t.Errorf("tags from --set should split on commas: %v", rec.Fields["tags"])
	}

	viaData := run(t, dir, "--json", "note", "create", "--data", `{"title":"From JSON","status":"published"}`)
	if viaData.code != 0 || !strings.Contains(viaData.stdout, "From JSON") {
		t.Errorf("create --data: %+v", viaData)
	}

	list := run(t, dir, "note", "list")
	if list.code != 0 || !strings.Contains(list.stdout, "Hello") || !strings.Contains(list.stdout, "From JSON") {
		t.Errorf("list: %+v", list)
	}
	listJSON := run(t, dir, "note", "list", "--json", "--limit", "1")
	var recs []any
	if json.Unmarshal([]byte(listJSON.stdout), &recs) != nil || len(recs) != 1 {
		t.Errorf("list --json --limit 1: %+v", listJSON)
	}

	got := run(t, dir, "note", "get", rec.ID)
	if got.code != 0 || !strings.Contains(got.stdout, "title: Hello") {
		t.Errorf("get: %+v", got)
	}

	upd := run(t, dir, "note", "update", rec.ID, "--set", "status=published", "--json")
	if upd.code != 0 || !strings.Contains(upd.stdout, `"status": "published"`) || !strings.Contains(upd.stdout, "Hello") {
		t.Errorf("update should merge: %+v", upd)
	}

	del := run(t, dir, "note", "delete", rec.ID)
	if del.code != 0 {
		t.Errorf("delete: %+v", del)
	}
	if again := run(t, dir, "note", "get", rec.ID); again.code != 1 || !strings.Contains(again.stderr, "not found") {
		t.Errorf("get after delete: %+v", again)
	}
}

// TestErrorsSayHowToFix checks the failure paths an agent must be able to
// recover from without human help.
func TestErrorsSayHowToFix(t *testing.T) {
	dir := initWorkspace(t)
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"note", "create", "--set", "status=bogus"}, "must be one of draft, published"},
		{[]string{"note", "create", "--set", "nope=1"}, "unknown field"},
		{[]string{"note", "create"}, "--set field=value"},
		{[]string{"note", "frobnicate"}, "use list, get, create, update, delete"},
		{[]string{"widget", "list"}, "unknown command or content type"},
		{[]string{"note", "update"}, "usage:"},
		{[]string{"chat"}, "usage: sameway chat"},
		{[]string{"chat", "hello"}, "no model configured"},
		{[]string{"component", "make"}, "usage: sameway component new"},
	}
	for _, c := range cases {
		r := run(t, dir, c.args...)
		if r.code != 1 || !strings.Contains(r.stderr, c.want) {
			t.Errorf("%v: code=%d stderr=%q (want %q)", c.args, r.code, r.stderr, c.want)
		}
	}
	r := run(t, dir, "note", "create", "--set", "status=bogus", "--json")
	var e struct{ Error string }
	if json.Unmarshal([]byte(r.stderr), &e) != nil || e.Error == "" {
		t.Errorf("--json errors must be JSON on stderr: %q", r.stderr)
	}
}

// TestComponentScaffoldIsValid checks a scaffolded component loads, renders,
// and shows up as a workspace component.
func TestComponentScaffoldIsValid(t *testing.T) {
	dir := initWorkspace(t)
	r := run(t, dir, "component", "new", "callout")
	if r.code != 0 {
		t.Fatalf("%+v", r)
	}
	for _, f := range []string{"manifest.json", "template.html", "style.css", "README.md"} {
		if _, err := os.Stat(filepath.Join(dir, "components", "callout", f)); err != nil {
			t.Errorf("scaffold missing %s", f)
		}
	}
	if again := run(t, dir, "component", "new", "callout"); again.code == 0 {
		t.Errorf("scaffolding over an existing component must fail")
	}
	check := run(t, dir, "check")
	if check.code != 0 {
		t.Errorf("check after scaffold: %+v", check)
	}
	d := run(t, dir, "describe", "--json")
	if !strings.Contains(d.stdout, `"name": "callout"`) || !strings.Contains(d.stdout, `"source": "workspace"`) {
		t.Errorf("describe should list the new workspace component")
	}
}

func TestCheckReportsBrokenExamples(t *testing.T) {
	dir := initWorkspace(t)
	run(t, dir, "component", "new", "bad")
	manifest := filepath.Join(dir, "components", "bad", "manifest.json")
	src, _ := os.ReadFile(manifest)
	os.WriteFile(manifest, bytes.Replace(src, []byte(`"props": { "text": "Hello" }`), []byte(`"props": { "wrong": 1 }`), 1), 0o644)
	r := run(t, dir, "check")
	if r.code != 1 || !strings.Contains(r.stderr, "component bad example default") {
		t.Errorf("check should name the broken example: %+v", r)
	}
}
