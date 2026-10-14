package main

import "testing"

// The browser suites run for any change that can alter what a page is
// drawn from, and only those are skipped that cannot; every edge the plan
// was found to have is a case here.
func TestNeedsBrowser(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		why   string
		files []string
		want  bool
	}{
		{"only what the assistant is told", []string{"internal/chat/layout_now.go", "internal/llm/llm.go", "internal/mcp/server.go"}, false},
		{"words and tests", []string{"docs/tests/canvas.md", "README.md", "internal/server/tabs_test.go"}, false},
		{"a component's README is words", []string{"design/components/tabs/README.md"}, false},
		{"the server draws pages", []string{"internal/chat/notice.go", "internal/server/tabs.go"}, true},
		{"a component", []string{"design/components/tabs/template.html"}, true},
		{"a golden example is what a component draws", []string{"design/components/tabs/examples/long.html"}, true},
		{"a starter record is Markdown the pages show", []string{"examples/workspaces/starter/content/note/welcome.md"}, true},
		{"the starter's words are its pages too", []string{"examples/workspaces/starter/README.md"}, true},
		{"a file moved out of the server is gone from there", []string{"internal/server/old.go", "internal/chat/old.go"}, true},
		{"the workflow itself", []string{".github/workflows/ci.yml"}, true},
		{"the modules", []string{"go.mod"}, true},
		{"the runner of the browser suites", []string{"tools/a11y-runner/site.mjs"}, true},
		{"the records a page lists", []string{"internal/records/glance.go"}, true},
		{"nothing changed is not nothing to check", nil, true},
		{"blank lines are not files", []string{"", "  ", "internal/chat/x.go"}, false},
		{"a Windows path", []string{`internal\server\tabs.go`}, true},
	} {
		if got := needsBrowser(c.files); got != c.want {
			t.Errorf("%s: %v gives %v, want %v", c.why, c.files, got, c.want)
		}
	}
}
