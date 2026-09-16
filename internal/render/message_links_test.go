package render_test

import (
	"strings"
	"testing"
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
	if !strings.Contains(got, "Linked content:") {
		t.Errorf("output should contain a visually-hidden heading for linked content:\n%s", got)
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
