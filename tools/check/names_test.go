package main

import "testing"

// Names or titles joined by a bare comma fail, whole or cut; other joins,
// schema.Names and tests pass.
func TestNamesJoinedByABareCommaFail(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	write(t, root, "internal/a/a.go", `package a

import "strings"

var names, titles, parts []string
var x = strings.Join(names, ", ")
var y = strings.Join(titles[:2], ", ")
var z = strings.Join(parts, ", ")
var w = strings.Join(names, "\n")
`)
	write(t, root, "internal/a/a_test.go", "package a\n\nimport \"strings\"\n\nvar v = strings.Join(names, \", \")\n")
	r := checkNameJoins(root)
	if len(r.problems) != 2 || !has(r.problems, "a.go:6 joins names") || !has(r.problems, "a.go:7 joins titles") {
		t.Errorf("problems = %q, want names and titles", r.problems)
	}
}
