package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// File and function size. Both are hard limits: a warning tier was tried
// and agents read it as permission, so a file over MaxLines fails, and the
// name rule below stops a split by size from passing it.
const (
	MaxLines     = 300 // a source file over this fails
	MaxFuncLines = 80  // a non-test Go function over this fails, unless in longFuncs
)

// sizeNames are file-name endings that say a file was split by size, not
// topic. A new one fails; the ones in splitByName are known.
var sizeNames = []string{"_more", "_extra", "_helpers"}

var sourceExt = map[string]bool{".go": true, ".css": true, ".html": true, ".js": true, ".mjs": true, ".json": true, ".yaml": true}

var skipDirs = map[string]bool{".git": true, "node_modules": true, "bin": true, "dist": true}

// report is what a check found: notes and warnings are printed, problems fail.
type report struct {
	notes    []string
	warnings []string
	problems []string
}

func (r *report) add(o report) {
	r.notes = append(r.notes, o.notes...)
	r.warnings = append(r.warnings, o.warnings...)
	r.problems = append(r.problems, o.problems...)
}

// walkSources calls fn with the slash path of every source file under root.
func walkSources(root string, fn func(rel string, full string)) {
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] && p != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !sourceExt[filepath.Ext(p)] {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			rel = p
		}
		fn(filepath.ToSlash(rel), p)
		return nil
	})
}

func checkFileSizes(root string) report {
	var r report
	walkSources(root, func(rel, full string) {
		data, err := os.ReadFile(full)
		if err != nil {
			return
		}
		lines := bytes.Count(data, []byte("\n"))
		if lines > MaxLines {
			r.problems = append(r.problems, fmt.Sprintf("%s has %d lines (max %d); split by topic: move a group of related functions into a file named after what it holds", rel, lines, MaxLines))
		}
	})
	return r
}

// checkFileNames refuses a new file named for being split by size.
func checkFileNames(root string) report {
	var r report
	seen := map[string]bool{}
	walkSources(root, func(rel, _ string) {
		if !strings.HasSuffix(rel, ".go") {
			return
		}
		stem := strings.TrimSuffix(strings.TrimSuffix(path.Base(rel), ".go"), "_test")
		for _, s := range sizeNames {
			if !strings.HasSuffix(stem, s) {
				continue
			}
			if splitByName[rel] {
				seen[rel] = true
				continue
			}
			r.problems = append(r.problems, fmt.Sprintf("%s is named %s, which says it was split by size; name it for what it does", rel, s))
		}
	})
	for _, rel := range sortedKeys(splitByName) {
		if !seen[rel] {
			r.warnings = append(r.warnings, fmt.Sprintf("%s is gone or renamed; take it out of splitByName in tools/check/debt.go", rel))
		}
	}
	return r
}

// checkFuncLengths fails a non-test function over MaxFuncLines unless it is
// in longFuncs, and fails one in longFuncs that has grown past its length
// there. Functions are keyed by package directory and name, so moving one
// between files of a package keeps its entry.
func checkFuncLengths(root string) report {
	var r report
	seen := map[string]bool{}
	fset := token.NewFileSet()
	walkSources(root, func(rel, full string) {
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return
		}
		f, err := parser.ParseFile(fset, full, nil, parser.SkipObjectResolution)
		if err != nil {
			r.problems = append(r.problems, fmt.Sprintf("%s does not parse: %v", rel, err))
			return
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			key := path.Dir(rel) + "." + funcName(fn)
			n := fset.Position(fn.End()).Line - fset.Position(fn.Pos()).Line + 1
			was, known := longFuncs[key]
			if known {
				seen[key] = true
			}
			switch {
			case known && n > was:
				r.problems = append(r.problems, fmt.Sprintf("%s: %s is %d lines, longer than the %d recorded in tools/check/debt.go; it may only shrink", rel, funcName(fn), n, was))
			case known && n < was:
				r.warnings = append(r.warnings, fmt.Sprintf("%s: %s is down to %d lines from %d; lower its entry in tools/check/debt.go (or remove it at %d or under)", rel, funcName(fn), n, was, MaxFuncLines))
			case !known && n > MaxFuncLines:
				r.problems = append(r.problems, fmt.Sprintf("%s: %s is %d lines (max %d); give its parts names of their own", rel, funcName(fn), n, MaxFuncLines))
			}
		}
	})
	for _, key := range sortedKeys(longFuncs) {
		if !seen[key] {
			r.warnings = append(r.warnings, fmt.Sprintf("%s is gone, renamed or moved; take it out of longFuncs in tools/check/debt.go (or move its key)", key))
		}
	}
	return r
}

// funcName is Name, or Recv.Name for a method.
func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	t := fn.Recv.List[0].Type
	if s, ok := t.(*ast.StarExpr); ok {
		t = s.X
	}
	switch x := t.(type) {
	case *ast.IndexExpr:
		t = x.X
	case *ast.IndexListExpr:
		t = x.X
	}
	if id, ok := t.(*ast.Ident); ok {
		return id.Name + "." + fn.Name.Name
	}
	return fn.Name.Name
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
