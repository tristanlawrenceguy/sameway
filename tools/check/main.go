// Command check lints the repository so AI-written contributions stay readable:
//
//   - no source file over MaxLines lines (agents read whole files before editing)
//   - every component folder has manifest.json, template.html, style.css, README.md, examples/
//   - design/tokens/tokens.css matches tokens.json
//
// Run from the repository root: go run ./tools/check
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sameway-dev/sameway/internal/tokens"
)

// MaxLines is the hard cap for any source file.
const MaxLines = 300

var sourceExt = map[string]bool{".go": true, ".css": true, ".html": true, ".js": true, ".mjs": true, ".json": true, ".yaml": true}

var skipDirs = map[string]bool{".git": true, "node_modules": true, "bin": true, "dist": true}

func main() {
	var problems []string
	problems = append(problems, checkFileSizes(".")...)
	problems = append(problems, checkComponents("design/components")...)
	problems = append(problems, checkTokens()...)
	for _, p := range problems {
		fmt.Fprintln(os.Stderr, "check:", p)
	}
	if len(problems) > 0 {
		fmt.Fprintf(os.Stderr, "check: %d problem(s)\n", len(problems))
		os.Exit(1)
	}
	fmt.Println("check: ok")
}

func checkFileSizes(root string) []string {
	var problems []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !sourceExt[filepath.Ext(path)] || strings.HasSuffix(path, "go.sum") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := bytes.Count(data, []byte("\n"))
		if lines > MaxLines {
			problems = append(problems, fmt.Sprintf("%s has %d lines (max %d); split it", filepath.ToSlash(path), lines, MaxLines))
		}
		return nil
	})
	return problems
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

func checkTokens() []string {
	src, err := os.ReadFile("design/tokens/tokens.json")
	if err != nil {
		return []string{"cannot read design/tokens/tokens.json: " + err.Error()}
	}
	want, err := tokens.Generate(src)
	if err != nil {
		return []string{err.Error()}
	}
	got, err := os.ReadFile("design/tokens/tokens.css")
	if err != nil || string(got) != want {
		return []string{"design/tokens/tokens.css is out of date; run `go run ./tools/tokens`"}
	}
	return nil
}
