package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"strconv"
	"strings"
)

// The debt list only goes down. Within one tree check can only see that a
// function has not outgrown its entry; raising the entry would pass. So
// with -base, check reads tools/check/debt.go as it was at that ref and
// fails a change that adds an entry or raises a recorded length. Lowering
// and removing entries is the point of the list.

const debtPath = "tools/check/debt.go"

// checkDebtAgainst compares the debt lists compiled into check with the
// ones in debt.go at base (or at its merge base with HEAD, when git can
// find one, so a branch behind main is not blamed for main's progress).
func checkDebtAgainst(base string) report {
	ref := base
	if out, err := exec.Command("git", "merge-base", base, "HEAD").Output(); err == nil {
		ref = strings.TrimSpace(string(out))
	}
	src, err := exec.Command("git", "show", ref+":"+debtPath).Output()
	if err != nil {
		return report{notes: []string{fmt.Sprintf("cannot read %s at %s (%v); the debt list was not compared with it", debtPath, base, err)}}
	}
	was, err := parseDebt(src)
	if err != nil {
		return report{notes: []string{fmt.Sprintf("cannot parse %s at %s (%v); the debt list was not compared with it", debtPath, base, err)}}
	}
	now := map[string]map[string]int{"longFuncs": longFuncs, "splitByName": {}, "storeWrites": {}}
	for k := range splitByName {
		now["splitByName"][k] = 1
	}
	for k := range storeWrites {
		now["storeWrites"][k] = 1
	}
	var r report
	for list := range now {
		if _, ok := was[list]; !ok {
			delete(now, list) // a list new since base has nothing to go down from
			r.notes = append(r.notes, fmt.Sprintf("%s is not in %s at %s; it was not compared", list, debtPath, base))
		}
	}
	r.add(compareDebt(was, now))
	return r
}

// compareDebt fails each entry of now that is not in was, or whose length
// is greater than in was.
func compareDebt(was, now map[string]map[string]int) report {
	var r report
	for _, list := range sortedKeys(now) {
		for _, key := range sortedKeys(now[list]) {
			n := now[list][key]
			old, ok := was[list][key]
			switch {
			case !ok && list == "longFuncs":
				r.problems = append(r.problems, fmt.Sprintf("%s: added to longFuncs in %s at %d lines; the list only goes down: shorten or split the function instead", key, debtPath, n))
			case !ok && list == "storeWrites":
				r.problems = append(r.problems, fmt.Sprintf("%s: added to storeWrites in %s; the list only goes down: write through records.Apply, ApplyOps or WriteAs instead", key, debtPath))
			case !ok:
				r.problems = append(r.problems, fmt.Sprintf("%s: added to %s in %s; the list only goes down: name the file for what it does instead", key, list, debtPath))
			case n > old:
				r.problems = append(r.problems, fmt.Sprintf("%s: raised from %d to %d in %s; shorten or split it instead", key, old, n, debtPath))
			}
		}
	}
	return r
}

// parseDebt reads the map literals of debt.go: longFuncs with their
// lengths, splitByName and storeWrites with 1 for each entry. A list not
// in src is not in what it returns.
func parseDebt(src []byte) (map[string]map[string]int, error) {
	f, err := parser.ParseFile(token.NewFileSet(), debtPath, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	known := map[string]bool{"longFuncs": true, "splitByName": true, "storeWrites": true}
	lists := map[string]map[string]int{}
	ast.Inspect(f, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 || len(spec.Values) != 1 {
			return true
		}
		name := spec.Names[0].Name
		lit, isLit := spec.Values[0].(*ast.CompositeLit)
		if !known[name] || !isLit {
			return true
		}
		list := map[string]int{}
		lists[name] = list
		for _, e := range lit.Elts {
			kv, ok := e.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			k, ok := kv.Key.(*ast.BasicLit)
			if !ok {
				continue
			}
			key, err := strconv.Unquote(k.Value)
			if err != nil {
				continue
			}
			list[key] = 1
			if v, ok := kv.Value.(*ast.BasicLit); ok && v.Kind == token.INT {
				list[key], _ = strconv.Atoi(v.Value)
			}
		}
		return false
	})
	return lists, nil
}
