package render_test

// The problem: who can fix it in sight, in the text colour so it never
// reads as a quiet empty line, and the detail behind a fold that looks
// like every other fold.

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

func renderProblem(t *testing.T, props map[string]any) string {
	t.Helper()
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}
	out, err := reg.Render("problem", props)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return string(out)
}

// TestAProblemFoldShowsItOpens: What is wrong carries the twisty, the one
// sign here that a summary opens something; without it the summary was
// grey words that did not look pressable.
func TestAProblemFoldShowsItOpens(t *testing.T) {
	out := renderProblem(t, map[string]any{"what": "This list", "text": `task has no field "owner"`})
	if !regexp.MustCompile(`<summary class="[^"]*\bsw-twisty\b[^"]*">What is wrong</summary>`).MatchString(out) {
		t.Errorf("What is wrong should carry the twisty:\n%s", out)
	}
	// The detail is what the server found, word for word: the capital is
	// drawn by the style, never written into what an agent reads.
	if !strings.Contains(out, `<p class="sw-problem__detail">task has no field &#34;owner&#34;</p>`) {
		t.Errorf("the detail should be the server's words as they are:\n%s", out)
	}
	// It is there when the page loads, so a live role would not be heard.
	if strings.Contains(out, "role=") || strings.Contains(out, "aria-live") {
		t.Errorf("a problem is found where it sits, not announced:\n%s", out)
	}
}

// TestAProblemIsNotMutedLikeAnEmptyLine: a quiet empty line is muted; the
// sentence that says a block is set up wrong is in the text colour, so the
// two never look the same.
func TestAProblemIsNotMutedLikeAnEmptyLine(t *testing.T) {
	css, err := fs.ReadFile(design.FS, "components/problem/style.css")
	if err != nil {
		t.Fatal(err)
	}
	said := regexp.MustCompile(`\.sw-problem__said \{[^}]*\}`).FindString(string(css))
	if said == "" || strings.Contains(said, "fg-muted") {
		t.Errorf("the sentence should be in the text colour, got %q", said)
	}
}
