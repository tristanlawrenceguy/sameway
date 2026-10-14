// Command scope says whether a change can alter what a page is drawn from,
// and so whether CI runs the browser suites for it. It reads the changed
// files, one a line, and prints true or false.
//
// The browser suites check what a page is drawn from: templates, css,
// scripts, the server, the components, and the starter workspace a test
// app serves. Seven browser jobs for a change that cannot touch any of
// those held runners other runs queued behind, and a fifteen-minute run
// took most of an hour. What cannot touch them: what the assistant is told
// (internal/chat, llm, mcp), words (docs, *.md) and tests (*_test.go).
// What a chat change can still change on a page, what it says, is read by
// the Go tests on every run (internal/server/every_page_reads_test.go).
//
//	git diff --name-only --no-renames HEAD^1 HEAD | go run ./tools/scope
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	var files []string
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		files = append(files, in.Text())
	}
	fmt.Println(needsBrowser(files))
}

// assistantOnly is what changes only what the assistant is told.
var assistantOnly = []string{"internal/chat/", "internal/llm/", "internal/mcp/", "docs/"}

// needsBrowser says whether any of files can change what a page is drawn
// from. Nothing changed, or nothing it can tell, runs them all.
func needsBrowser(files []string) bool {
	seen := false
	for _, f := range files {
		f = strings.TrimSpace(strings.ReplaceAll(f, `\`, "/"))
		if f == "" {
			continue
		}
		seen = true
		if !pageless(f) {
			return true
		}
	}
	return !seen
}

// pageless says a file cannot change what a page is drawn from.
func pageless(f string) bool {
	// A workspace's records are Markdown (content/<type>/*.md): under
	// examples/, a .md is what the starter's pages show.
	if strings.HasPrefix(f, "examples/") {
		return false
	}
	if strings.HasSuffix(f, "_test.go") || strings.HasSuffix(f, ".md") {
		return true
	}
	for _, dir := range assistantOnly {
		if strings.HasPrefix(f, dir) {
			return true
		}
	}
	return false
}
