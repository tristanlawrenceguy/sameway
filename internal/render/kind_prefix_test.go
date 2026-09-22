package render_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestOtherKindsKeepKindPrefix checks that info, warning, and danger alerts
// still include their kind text prefixes ("Note:", "Warning:", "Error:") so
// only success is changed. Acceptance item 1 (negative check).
func TestOtherKindsKeepKindPrefix(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		kind   string
		prefix string
	}{
		{"info", "Note:"},
		{"warning", "Warning:"},
		{"danger", "Error:"},
	} {
		got, err := reg.Render("alert", map[string]any{
			"kind":    tc.kind,
			"title":   "A title",
			"message": "some message",
		})
		if err != nil {
			t.Fatalf("%s: render: %v", tc.kind, err)
		}

		out := string(got)
		if !contains(out, tc.prefix) {
			t.Errorf("kind=%s alert should contain %q; got:\n%s", tc.kind, tc.prefix, out)
		}
		// Also ensure the kind span exists for non-success kinds.
		if !contains(out, `<span class="sw-alert__kind">`) {
			t.Errorf("kind=%s alert should have sw-alert__kind span; got:\n%s", tc.kind, out)
		}
	}
}
