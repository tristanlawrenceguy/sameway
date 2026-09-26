package render_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestMessageWithLinks renders an assistant message that includes links, which is what
// the chat code will do once it computes creation links from changes.  This test fails
// today because the message component has no `links` prop declared and rejects unknown
// properties; after adding the optional array prop it should render cleanly with a
// .sw-message__links list between content and changes.
func TestMessageWithLinks(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "I added two items.",
		"time":    "14:05",
		"links": []any{
			map[string]any{"href": "/t/note/abc", "label": "a note"},
			map[string]any{"href": "/canvas/h1", "label": "heading"},
		},
	})
	if err != nil {
		t.Fatalf("message with links should not error: %v", err)
	}

	got := string(out)
	if !strings.Contains(got, `class="sw-message__links"`) {
		t.Errorf("output missing .sw-message__links list:\n%s", got)
	}
	if !strings.Contains(got, "/t/note/abc") {
		t.Errorf("output should contain the link href:\n%s", got)
	}
	if !strings.Contains(got, `<p class="sw-message__label">Linked</p>`) {
		t.Errorf("output should label the linked content where it can be seen:\n%s", got)
	}
}

// TestMessageWithChangesAndLinks renders an assistant message that has both changes and
// links, proving the two sections coexist correctly in the output.  This is the shape the
// chat code will use once it computes creation links from each change record.
func TestMessageWithChangesAndLinks(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "I added two items.",
		"time":    "14:05",
		"changes": []any{
			map[string]any{"action": "added", "component": "note"},
		},
		"links": []any{
			map[string]any{"href": "/t/note/abc", "label": "a note"},
		},
	})
	if err != nil {
		t.Fatalf("message with both changes and links should not error: %v", err)
	}

	got := string(out)
	if !strings.Contains(got, `class="sw-message__links"`) {
		t.Errorf("output missing .sw-message__links list:\n%s", got)
	}
	if !strings.Contains(got, `class="sw-message__changes"`) {
		t.Errorf("output should still contain the changes section:\n%s", got)
	}
}

// TestMessageGoldenWithLinks checks that the "with links" example exists in the message
// component's manifest and produces valid HTML.  This is covered by the general golden
// test but adds an explicit assertion here so the acceptance item about inline links is
// pinned down directly: if a developer removes the links section from the template, this
// test will catch it even without re-running UPDATE_GOLDEN=1.
func TestMessageGoldenWithLinks(t *testing.T) {
	reg := builtins(t)

	// Find the message component and look for an example named "with links"
	var found bool
	for _, c := range reg.Components() {
		if c.Manifest.Name != "message" {
			continue
		}
		for _, ex := range c.Manifest.Examples {
			if ex.Name == "with links" {
				found = true

				out, err := c.Render(ex.Props)
				if err != nil {
					t.Errorf("example %s should not error: %v", ex.Name, err)
					continue
				}

				got := string(out)
				if !strings.Contains(got, `class="sw-message__links"`) {
					t.Errorf("with-links example missing .sw-message__links:\n%s", got)
				}
				if !strings.Contains(got, "I added two items.") {
					t.Errorf("with-links example should contain the content text:\n%s", got)
				}
			}
		}
		break
	}

	if !found {
		t.Error("message component manifest must include a 'with links' example")
	}
}

// TestMessageInternalLinkRendersAsAnchor renders an assistant message whose
// content contains an internal path like /t/note/abc123 and asserts it becomes
// a clickable anchor tag with class "sw-link". Acceptance item 1.
func TestMessageInternalLinkRendersAsAnchor(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "See the note at /t/note/abc123 for details.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)
	want := `<a class="sw-link" href="/t/note/abc123">/t/note/abc123</a>`
	if !strings.Contains(got, want) {
		t.Errorf("internal path should become a clickable link\nwant: %s\ngot:\n%s", want, got)
	}
}

// TestMessageFileLinkRendersAsAnchor renders an assistant message whose content
// contains /t/file/def456 and asserts it becomes a clickable anchor tag.
// Acceptance item 2.
func TestMessageFileLinkRendersAsAnchor(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "I created a file at /t/file/def456.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)
	want := `<a class="sw-link" href="/t/file/def456">/t/file/def456</a>`
	if !strings.Contains(got, want) {
		t.Errorf("internal file path should become a clickable link\nwant: %s\ngot:\n%s", want, got)
	}
}

// TestMessageExternalURLRendersAsAnchor renders an assistant message whose
// content contains https://example.com and asserts it becomes a clickable anchor.
// Acceptance item 3.
func TestMessageExternalURLRendersAsAnchor(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "Check https://example.com for more.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)
	want := `<a class="sw-link" href="https://example.com">example.com</a>`
	if !strings.Contains(got, want) {
		t.Errorf("external URL should become a clickable link\nwant: %s\ngot:\n%s", want, got)
	}
}

// TestMessagePlainTextUnchanged renders an assistant message with no URLs and
// asserts the content is plain text — no anchor tags are injected. Acceptance
// item 4.
func TestMessagePlainTextUnchanged(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "No links here.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)
	if strings.Contains(got, `<a class="sw-link"`) {
		t.Errorf("message with no URLs must not contain anchor tags:\ngot:\n%s", got)
	}
	if !strings.Contains(got, "<p>No links here.</p>") {
		t.Errorf("plain text should appear as-is in a <p> tag:\ngot:\n%s", got)
	}
}

// TestMessageTemplateUsesLinkify asserts that the message component template
// calls linkify on paragraph content, which is how acceptance item 5 is met.
// This test reads the template source directly so it does not depend on a
// particular render output — if the developer adds linkify to Funcs but
// forgets to wire it into the template, this still fails.
func TestMessageTemplateUsesLinkify(t *testing.T) {
	reg := builtins(t)

	var c *render.Component
	for _, comp := range reg.Components() {
		if comp.Manifest.Name == "message" {
			c = comp
			break
		}
	}
	if c == nil {
		t.Fatal("no message component registered")
	}

	templateSrc, err := c.ReadFile("template.html")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}

	src := string(templateSrc)
	if !strings.Contains(src, "linkify") {
		t.Errorf("message template must call linkify on paragraph content to make inline URLs clickable\nwant: {{linkify .}} in the content range block\ngot:\n%s", src)
	}
}

// TestMessageMixedLinksOrdered renders a paragraph containing both an external
// URL and an internal path where the URL appears first, verifying anchors are
// emitted in text order (not regex-loop order). Addresses reviewer finding 3.
func TestMessageMixedLinksOrdered(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "Visit https://example.com or see /t/note/abc123.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	// Both anchors must be present.
	wantURL := `<a class="sw-link" href="https://example.com">example.com</a>`
	wantPath := `<a class="sw-link" href="/t/note/abc123">/t/note/abc123</a>`
	if !strings.Contains(got, wantURL) {
		t.Errorf("missing external URL anchor:\ngot:\n%s", got)
	}
	if !strings.Contains(got, wantPath) {
		t.Errorf("missing internal path anchor:\ngot:\n%s", got)
	}

	// The URL anchor must appear before the internal path anchor (text order).
	urlPos := strings.Index(got, wantURL)
	pathPos := strings.Index(got, wantPath)
	if urlPos < 0 || pathPos < 0 {
		t.Fatalf("could not find both anchors to check order:\ngot:\n%s", got)
	}
	if urlPos > pathPos {
		t.Errorf("anchors must appear in text order; URL anchor should come before internal path\nwant: %s … %s\ngot:\n%s", wantURL, wantPath, got)
	}
}

// TestMessageOverlappingURLAndPath renders content where an external URL
// contains a substring matching /t/<type>/<id>, verifying only one anchor for
// the full URL is emitted (no nested anchors). Addresses reviewer finding 4.
func TestMessageOverlappingURLAndPath(t *testing.T) {
	reg := builtins(t)

	out, err := reg.Render("message", map[string]any{
		"role":    "assistant",
		"content": "See https://example.com/t/note/abc123.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(out)

	// The full URL anchor must be present.
	wantURL := `<a class="sw-link" href="https://example.com/t/note/abc123">example.com/t/note/abc123</a>`
	if !strings.Contains(got, wantURL) {
		t.Errorf("full URL anchor missing:\ngot:\n%s", got)
	}

	// There must be exactly one <a class="sw-link"> in the output — no nested
	// or overlapping anchors for the /t/note/abc123 substring.
	count := strings.Count(got, `<a class="sw-link"`)
	if count != 1 {
		t.Errorf("expected exactly one sw-link anchor, got %d:\ngot:\n%s", count, got)
	}
}
