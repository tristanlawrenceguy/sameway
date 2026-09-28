package prose_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/look"
	"github.com/tristanlawrenceguy/sameway/internal/prose"
)

// Markdown becomes the structure a page needs: headings that fit the
// outline, lists, emphasis, links, code, and tables with captions; nothing
// raw survives and nothing can run.
func TestMarkdownBecomesStructureThatFitsThePage(t *testing.T) {
	out := string(prose.Render("# Plan\n\nSome *words* and a [link](/t/note).\n\n## Steps\n\n- one\n- two\n\nTable: What to buy\n\n| Item | Count |\n|---|---|\n| Milk | 2 |\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n```\ncode\n```\n", 2))
	for _, want := range []string{"<h2>Plan</h2>", "<h3>Steps</h3>", "<em>words</em>", `<a href="/t/note">link</a>`, "<ul>", "<li>one</li>", `<div class="sw-table-wrap" role="region" tabindex="0" aria-label="What to buy"><table class="sw-table"><caption>What to buy</caption>`, `<table class="sw-table"><caption class="sw-visually-hidden">Table</caption>`, "</table></div>", "<pre><code>code"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Table: What to buy") {
		t.Error("the caption line should become the caption, not stay a paragraph")
	}
	if o, _ := look.Fragment(out); len(o.Problems) != 0 {
		t.Errorf("rendered prose should read cleanly, got %v", o.Problems)
	}

	// Inside a card whose title is an h3, a "#" is an h4.
	if deep := string(prose.Render("# Inside", 4)); !strings.Contains(deep, "<h4>Inside</h4>") {
		t.Errorf("headings should shift to the base level, got %s", deep)
	}
	if capped := string(prose.Render("### Far down", 6)); !strings.Contains(capped, "<h6>") {
		t.Errorf("levels stop at six, got %s", capped)
	}

	// Nothing raw, nothing that runs.
	hostile := string(prose.Render("<script>alert(1)</script> [x](javascript:alert(1)) <img src=x onerror=alert(1)>", 2))
	if strings.Contains(hostile, "<script>") || strings.Contains(hostile, "javascript:") || strings.Contains(hostile, "onerror") {
		t.Errorf("markup in markdown must be inert, got %s", hostile)
	}
	if plain := string(prose.Render("Just words.\n\nMore words.", 2)); !strings.Contains(plain, "<p>Just words.</p>\n<p>More words.</p>") {
		t.Errorf("plain paragraphs stay paragraphs, got %s", plain)
	}
}
