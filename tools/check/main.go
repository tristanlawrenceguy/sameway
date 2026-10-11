// Command check lints the repository so AI-written contributions stay readable:
//
//   - a source file over MaxLines lines fails
//   - a non-test Go function over MaxFuncLines fails, unless it is one of the
//     long functions recorded in debt.go, which may only shrink
//   - no new Go file named *_more, *_extra or *_helpers (split by size, not topic)
//   - every component folder has manifest.json, template.html, style.css, README.md, examples/
//   - design/tokens/tokens.css matches tokens.json
//   - a direct write to the store outside store and records fails, unless
//     storeWrites in debt.go lists it with its reason (writes.go)
//   - with -base REF: tools/check/debt.go only goes down from REF (no entry
//     added, no recorded length raised)
//
// Run from the repository root: go run ./tools/check [-base origin/main]
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tristanlawrenceguy/sameway/internal/tokens"
)

func main() {
	base := flag.String("base", "", "git ref to compare tools/check/debt.go with; the debt list may only go down from it")
	flag.Parse()
	r := run(".")
	if *base != "" {
		r.add(checkDebtAgainst(*base))
	}
	for _, n := range r.notes {
		fmt.Println("check: note:", n)
	}
	for _, w := range r.warnings {
		fmt.Println("check: warning:", w)
	}
	for _, p := range r.problems {
		fmt.Fprintln(os.Stderr, "check:", p)
	}
	if len(r.problems) > 0 {
		fmt.Fprintf(os.Stderr, "check: %d problem(s)\n", len(r.problems))
		os.Exit(1)
	}
	fmt.Println("check: ok")
}

// run checks the repository at root.
func run(root string) report {
	var r report
	r.add(checkFileSizes(root))
	r.add(checkFileNames(root))
	r.add(checkFuncLengths(root))
	r.add(checkStoreWrites(root, storeWrites))
	r.problems = append(r.problems, checkComponents(filepath.Join(root, "design/components"))...)
	r.problems = append(r.problems, checkTokens(root)...)
	return r
}

func checkComponents(root string) []string {
	var problems []string
	entries, err := os.ReadDir(root)
	if err != nil {
		return []string{"cannot read " + root + ": " + err.Error()}
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		for _, required := range []string{"manifest.json", "template.html", "style.css", "README.md", "examples"} {
			if _, err := os.Stat(filepath.Join(dir, required)); err != nil {
				problems = append(problems, fmt.Sprintf("component %s is missing %s", e.Name(), required))
			}
		}
		manifest, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
		if err != nil {
			continue
		}
		for _, key := range []string{`"props"`, `"a11y"`, `"machine"`, `"examples"`} {
			if !bytes.Contains(manifest, []byte(key)) {
				problems = append(problems, fmt.Sprintf("component %s manifest has no %s section", e.Name(), key))
			}
		}
	}
	return problems
}

func checkTokens(root string) []string {
	src, err := os.ReadFile(filepath.Join(root, "design/tokens/tokens.json"))
	if err != nil {
		return []string{"cannot read design/tokens/tokens.json: " + err.Error()}
	}
	want, err := tokens.Generate(src)
	if err != nil {
		return []string{err.Error()}
	}
	got, err := os.ReadFile(filepath.Join(root, "design/tokens/tokens.css"))
	if err != nil || string(got) != want {
		return []string{"design/tokens/tokens.css is out of date; run `go run ./tools/tokens`"}
	}
	return nil
}
