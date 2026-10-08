package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// goFunc is a Go function of exactly n lines, its signature to its brace.
func goFunc(name string, n int) string {
	return "func " + name + "() {\n" + strings.Repeat("\t_ = 1\n", n-2) + "}\n"
}

func has(list []string, part string) bool {
	for _, s := range list {
		if strings.Contains(s, part) {
			return true
		}
	}
	return false
}

func TestFileOverWarnLinesWarnsAndOverMaxLinesFails(t *testing.T) {
	root := t.TempDir()
	write(t, root, "ok.css", strings.Repeat("a\n", WarnLines))
	write(t, root, "warn.css", strings.Repeat("a\n", WarnLines+1))
	write(t, root, "fail.css", strings.Repeat("a\n", MaxLines+1))
	write(t, root, "node_modules/big.js", strings.Repeat("a\n", MaxLines+1))
	r := checkFileSizes(root)
	if len(r.warnings) != 1 || !has(r.warnings, "warn.css has 301 lines") {
		t.Errorf("warnings = %q, want warn.css alone", r.warnings)
	}
	if len(r.problems) != 1 || !has(r.problems, "fail.css has 401 lines") {
		t.Errorf("problems = %q, want fail.css alone", r.problems)
	}
}

func TestNewLongFunctionFailsAndTestFilesAreExempt(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a/a.go", "package a\n\n"+goFunc("short", MaxFuncLines)+goFunc("long", MaxFuncLines+1))
	write(t, root, "a/a_test.go", "package a\n\n"+goFunc("longTest", MaxFuncLines*3))
	write(t, root, "a/m.go", "package a\n\ntype T struct{}\n\nfunc (t *T) "+strings.TrimPrefix(goFunc("Method", MaxFuncLines+5), "func "))
	defer swap(&longFuncs, map[string]int{})()
	r := checkFuncLengths(root)
	if len(r.problems) != 2 || !has(r.problems, "a/a.go: long is 81 lines") || !has(r.problems, "a/m.go: T.Method is 85 lines") {
		t.Errorf("problems = %q, want long and T.Method", r.problems)
	}
}

func TestRecordedLongFunctionMayOnlyShrink(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a/a.go", "package a\n\n"+goFunc("same", 100)+goFunc("grown", 101)+goFunc("shrunk", 90))
	defer swap(&longFuncs, map[string]int{"a.same": 100, "a.grown": 100, "a.shrunk": 100, "a.gone": 120})()
	r := checkFuncLengths(root)
	if len(r.problems) != 1 || !has(r.problems, "grown is 101 lines, longer than the 100 recorded") {
		t.Errorf("problems = %q, want grown alone", r.problems)
	}
	if !has(r.warnings, "shrunk is down to 90 lines from 100") || !has(r.warnings, "a.gone is gone") || len(r.warnings) != 2 {
		t.Errorf("warnings = %q, want shrunk and gone", r.warnings)
	}
}

func TestFilesNamedForSizeFailUnlessRecorded(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"a/undo_more.go", "a/views_helpers.go", "a/props_extra_test.go", "a/said_test_helpers_test.go", "a/old_more.go", "a/moreover.go", "a/helpers.go"} {
		write(t, root, rel, "package a\n")
	}
	defer swap(&splitByName, map[string]bool{"a/old_more.go": true, "a/renamed_extra.go": true})()
	r := checkFileNames(root)
	for _, want := range []string{"a/undo_more.go", "a/views_helpers.go", "a/props_extra_test.go", "a/said_test_helpers_test.go"} {
		if !has(r.problems, want) {
			t.Errorf("problems = %q, want %s", r.problems, want)
		}
	}
	if len(r.problems) != 4 {
		t.Errorf("problems = %q, want four (old_more.go is recorded; moreover.go and helpers.go are names of their own)", r.problems)
	}
	if len(r.warnings) != 1 || !has(r.warnings, "a/renamed_extra.go is gone") {
		t.Errorf("warnings = %q, want renamed_extra.go", r.warnings)
	}
}

func swap[V any](p *map[string]V, m map[string]V) func() {
	old := *p
	*p = m
	return func() { *p = old }
}
