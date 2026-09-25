package render_test

// Tests for keyboard accessibility of the card component (task 0198).
// Verifies that a linked card renders a native <a> with visible href inside
// a heading tag, and has no tabindex="-1" — so a tab user can reach it.

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestCardKeyboard verifies acceptance items 3–5: when a card has an href,
// the title is a native <a> element with a visible href inside a heading tag,
// and no tabindex="-1" appears in the output. This ensures a keyboard user
// can reach the link via Tab and activate it natively.
func TestCardKeyboard(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	out, err := reg.Render("card", map[string]any{
		"title": "Read the architecture",
		"href":  "/t/note/test123",
	})
	if err != nil {
		t.Fatalf("render card: %v", err)
	}

	got := string(out)

	// 1. The output contains a native <a> element with the href attribute visible
	//    in the markup — keyboard users can activate it by Tabbing to it and
	//    pressing Enter (Acceptance item 3).
	if !strings.Contains(got, `<a href="/t/note/test123">`) {
		t.Errorf("card with href should render a native <a> with visible href;\ngot:\n%s", got)
		return
	}

	// 2. No tabindex="-1" anywhere in the card — the link is naturally tabbable
	//    because it sits in normal flow (Acceptance item 4).
	if strings.Contains(got, `tabindex="-1"`) {
		t.Errorf("card must not use tabindex=\"-1\" (blocks keyboard focus);\ngot:\n%s", got)
	}

	// 3. The <a> element is inside a heading tag (<h2>–<h6>). The default level
	//    is 3, so the output should contain <h3> wrapping the link (Acceptance item 5).
	if !strings.Contains(got, "<h3") {
		t.Errorf("card with default level should wrap title in a heading tag; expected <h3>, got:\n%s", got)
	}

	// The heading must also carry the data-prop="title" attribute so agents can
	// identify it. This is not a keyboard concern but verifies we rendered the
	// correct element.
	if !strings.Contains(got, `data-prop="title"`) {
		t.Errorf("card title heading should have data-prop=\"title\";\ngot:\n%s", got)
	}

	// 4. Verify the card root has its component marker for contract tests too.
	if !strings.Contains(got, `data-component="card"`) {
		t.Errorf("card root should carry data-component=\"card\";\ngot:\n%s", got)
	}
}
