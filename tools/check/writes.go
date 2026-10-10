package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strconv"
	"strings"
)

// A record is written one way from every way in: as a change's ops, by
// records.WriteAs or Apply, which write, keep what was there and log it as
// whoever did it. Code that writes the store itself has to do all three
// and can forget one, as the API's file without content once forgot the
// log, so it could never be undone. This check finds every call that
// writes the store directly, in every package but store and records, and
// fails one that storeWrites in debt.go does not list with its reason.
// It used to be a test in internal/server that read only server, cli, mcp
// and app, so the packages split out of server since (media, exchange,
// connect) and chat were never read.

// storeWriters are the store's methods that write a record.
var storeWriters = map[string]bool{"Create": true, "Update": true, "Delete": true, "DeleteAll": true, "Restore": true, "Put": true}

// writeRoots are the folders read, and writeOwners the packages that are
// the one way in, so are not.
var (
	writeRoots  = []string{"internal", "cmd", "tools"}
	writeOwners = map[string]bool{"internal/store": true, "internal/records": true}
)

type goFile struct {
	rel  string
	file *ast.File
}

// checkStoreWrites fails a direct write to the store not in storeWrites,
// and an entry there that no longer writes.
func checkStoreWrites(root string, allowed map[string]string) report {
	var r report
	found := storeWritesIn(root)
	for _, key := range sortedKeys(found) {
		if allowed[key] == "" {
			r.problems = append(r.problems, fmt.Sprintf("%s writes the store itself: write records through records.Apply, ApplyOps or WriteAs, or say why it does not in storeWrites in %s", key, debtPath))
		}
	}
	for _, key := range sortedKeys(allowed) {
		if !found[key] {
			r.problems = append(r.problems, fmt.Sprintf("%s no longer writes the store itself; take it out of storeWrites in %s", key, debtPath))
		}
	}
	return r
}

// storeWritesIn is every function that writes the store directly, keyed
// by its file and name ("internal/chat/access.go Service.setAccess").
func storeWritesIn(root string) map[string]bool {
	byDir := map[string][]goFile{}
	for _, top := range writeRoots {
		walkSources(path.Join(root, top), func(rel, full string) {
			rel = top + "/" + rel
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || writeOwners[path.Dir(rel)] {
				return
			}
			src, err := os.ReadFile(full)
			if err != nil {
				return
			}
			f, err := parser.ParseFile(token.NewFileSet(), rel, src, parser.SkipObjectResolution)
			if err != nil {
				return
			}
			byDir[path.Dir(rel)] = append(byDir[path.Dir(rel)], goFile{rel, f})
		})
	}
	found := map[string]bool{}
	for _, files := range byDir {
		names := storeNames(files)
		for _, gf := range files {
			for _, d := range gf.file.Decls {
				for name, body := range decls(d) {
					if writesStore(body, names) {
						found[gf.rel+" "+name] = true
					}
				}
			}
		}
	}
	return found
}

// storeNames are the names a package gives a *store.Store: its fields,
// parameters and variables of that type, and what is assigned one.
// Store is always one, the field every service and the app hold it in.
func storeNames(files []goFile) map[string]bool {
	names := map[string]bool{"Store": true}
	for _, gf := range files {
		pkg := storeImport(gf.file)
		if pkg == "" {
			continue
		}
		ast.Inspect(gf.file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Field:
				if isStoreType(n.Type, pkg) {
					for _, id := range n.Names {
						names[id.Name] = true
					}
				}
			case *ast.ValueSpec:
				if isStoreType(n.Type, pkg) {
					for _, id := range n.Names {
						names[id.Name] = true
					}
				}
			case *ast.AssignStmt:
				for i, rhs := range n.Rhs {
					if sel, ok := rhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "Store" && i < len(n.Lhs) {
						if id, ok := n.Lhs[i].(*ast.Ident); ok {
							names[id.Name] = true
						}
					}
				}
			}
			return true
		})
	}
	return names
}

// storeImport is the name a file imports internal/store under, or "".
func storeImport(f *ast.File) string {
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		if !strings.HasSuffix(p, "/internal/store") {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "store"
	}
	return ""
}

func isStoreType(e ast.Expr, pkg string) bool {
	star, ok := e.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == pkg && sel.Sel.Name == "Store"
}

// decls are a declaration's names and what runs in each: a function's
// body, or a package variable's value, such as an Op whose Run is a func.
func decls(d ast.Decl) map[string]ast.Node {
	out := map[string]ast.Node{}
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Body != nil {
			out[funcName(d)] = d.Body
		}
	case *ast.GenDecl:
		for _, s := range d.Specs {
			if vs, ok := s.(*ast.ValueSpec); ok && d.Tok == token.VAR && len(vs.Names) > 0 {
				out[vs.Names[0].Name] = vs
			}
		}
	}
	return out
}

// writesStore says whether a body calls a store's write method on
// something the package names a store.
func writesStore(body ast.Node, names map[string]bool) bool {
	hit := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || hit {
			return !hit
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !storeWriters[sel.Sel.Name] {
			return true
		}
		switch x := sel.X.(type) {
		case *ast.Ident:
			hit = names[x.Name]
		case *ast.SelectorExpr:
			hit = names[x.Sel.Name]
		}
		return true
	})
	return hit
}
