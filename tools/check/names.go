package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strings"
)

// Names and titles are never joined by a bare comma. A name can have a
// comma of its own (an entry is "Running miles, 5 miles"), and three such
// names joined by commas read as six (crew finding 0748). The crew fixed
// that in the log, then found it again in the next place that joined
// names; schema.Names and schema.AndNames set them apart by semicolons
// when one has a comma, and this finds a join that does not use them.

// checkNameJoins fails strings.Join(<names or titles>, ", ") under
// internal, outside tests.
func checkNameJoins(root string) report {
	var r report
	walkSources(path.Join(root, "internal"), func(rel, full string) {
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return
		}
		src, err := os.ReadFile(full)
		if err != nil {
			return
		}
		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, rel, src, parser.SkipObjectResolution)
		if err != nil {
			return
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if what := namesJoined(n); what != "" {
				r.problems = append(r.problems, fmt.Sprintf("internal/%s:%d joins %s with a bare comma; a name can have a comma of its own, so use schema.Names or schema.AndNames", rel, fs.Position(n.Pos()).Line, what))
			}
			return true
		})
	})
	return r
}

// namesJoined is the names a call joins by ", ", when it is a join of
// names or titles; "" otherwise.
func namesJoined(n ast.Node) string {
	call, ok := n.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 || stringLit(call.Args[1]) != ", " {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Join" {
		return ""
	}
	if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "strings" {
		return ""
	}
	var id *ast.Ident
	switch a := call.Args[0].(type) {
	case *ast.Ident:
		id = a
	case *ast.SliceExpr:
		id, _ = a.X.(*ast.Ident)
	}
	if id == nil {
		return ""
	}
	low := strings.ToLower(id.Name)
	if strings.Contains(low, "name") || strings.Contains(low, "title") {
		return id.Name
	}
	return ""
}
