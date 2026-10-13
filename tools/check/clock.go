package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// The time it is for the person comes from the app's clock
// (app.Options.Clock: s.now() in the server, Book.Now, Workspace.now,
// Store.Clock), which a test fixes, so a page, a reminder or a copy made
// says the same time whatever the test machine's clock says. A
// time.Now() in the code that serves pages, the assistant, records,
// blocks, the store and the schema reads the computer's clock instead.
// The computer's zone (time.Local, t.Local()) is the same mistake: the
// person's days are in the zone of the app's clock (now.Location()),
// which is not this computer's when the workspace is hosted elsewhere.
// Where real time is meant (how long a request waits, who is here just
// now) the line says so with a `// wall clock: <reason>` comment. The
// files that still read it unexplained are listed in wallClock in
// debt.go, with how many times, and that list only goes down.

// clockDirs are where time.Now() and time.Local must be the app's clock
// and its zone, or say why not.
var clockDirs = []string{"internal/server/", "internal/chat/", "internal/automate/", "internal/records/", "internal/blocks/", "internal/store/", "internal/schema/"}

const wallClockMark = "wall clock:"

// wallClockCalls counts, for each non-test Go file under clockDirs, its
// uses of time.Now, time.Local and t.Local() that no `// wall clock:`
// comment on the same line explains. Files with none are left out.
func wallClockCalls(root string) (map[string]int, []string) {
	found := map[string]int{}
	var problems []string
	fset := token.NewFileSet()
	walkSources(root, func(rel, full string) {
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || !underClockDirs(rel) {
			return
		}
		f, err := parser.ParseFile(fset, full, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s does not parse: %v", rel, err))
			return
		}
		marked := map[int]bool{}
		for _, g := range f.Comments {
			for _, c := range g.List {
				if strings.Contains(c.Text, wallClockMark) {
					marked[fset.Position(c.Pos()).Line] = true
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok && len(call.Args) == 0 {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Local" && !marked[fset.Position(sel.Pos()).Line] {
					found[rel]++ // t.Local()
				}
			}
			sel, ok := n.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Now" && sel.Sel.Name != "Local" {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "time" && !marked[fset.Position(sel.Pos()).Line] {
				found[rel]++
			}
			return true
		})
	})
	return found, problems
}

func underClockDirs(rel string) bool {
	for _, d := range clockDirs {
		if strings.HasPrefix(rel, d) {
			return true
		}
	}
	return false
}

// checkWallClock fails a file that reads time.Now() or time.Local
// unexplained more often than known records for it (known is wallClock in
// debt.go).
func checkWallClock(root string, known map[string]int) report {
	found, problems := wallClockCalls(root)
	r := report{problems: problems}
	for _, rel := range sortedKeys(found) {
		n := found[rel]
		was, ok := known[rel]
		switch {
		case !ok:
			r.problems = append(r.problems, fmt.Sprintf("%s: time.Now() or time.Local read %d time(s); use the app's clock (s.now(), Book.Now, Store.Now) and its zone (now.Location()), or say why it must be real time with `// wall clock: <reason>` on its line", rel, n))
		case n > was:
			r.problems = append(r.problems, fmt.Sprintf("%s: time.Now() or time.Local read %d times, more than the %d recorded in wallClock in tools/check/debt.go; use the app's clock (s.now(), Book.Now, Store.Now) and its zone (now.Location()), or say why it must be real time with `// wall clock: <reason>`", rel, n, was))
		case n < was:
			r.warnings = append(r.warnings, fmt.Sprintf("%s: time.Now() down to %d from %d; lower its entry in wallClock in tools/check/debt.go", rel, n, was))
		}
	}
	for _, rel := range sortedKeys(known) {
		if found[rel] == 0 {
			r.warnings = append(r.warnings, fmt.Sprintf("%s reads time.Now() unexplained no more; take it out of wallClock in tools/check/debt.go", rel))
		}
	}
	return r
}
