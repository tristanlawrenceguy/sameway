package render_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
	"github.com/tristanlawrenceguy/sameway/internal/render"
)

// TestSuccessAlertNoKindPrefix checks that a success alert does NOT include
// the "Success:" kind label prefix in its rendered output. The green styling,
// role="status", and checkmark icon already communicate success; only info,
// warning, and danger kinds show their text prefixes. Acceptance item 1.
func TestSuccessAlertNoKindPrefix(t *testing.T) {
	reg := render.New()
	if err := reg.LoadFS(design.FS, "components", "builtin"); err != nil {
		t.Fatal(err)
	}

	got, err := reg.Render("alert", map[string]any{
		"kind":    "success",
		"title":   "Changes saved",
		"message": "Your edits were applied.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	out := string(got)

	// The word "Success:" must NOT appear anywhere in the output.
	if contains(out, "Success:") {
		t.Errorf("success alert should not contain \"Success:\" prefix;\ngot:\n%s\nwant: no \"Success:\" text", out)
	}

	// The success kind span must also be absent entirely (no empty <span>).
	if contains(out, `<span class="sw-alert__kind">`) {
		t.Errorf("success alert should not render a sw-alert__kind span;\ngot:\n%s\nwant: no kind span at all", out)
	}

	// The remaining success signals must still be present.
	if !contains(out, `data-kind="success"`) {
		t.Error("output missing data-kind=\"success\"")
	}
	if !contains(out, "Changes saved") {
		t.Errorf("title text \"Changes saved\" should appear; got:\n%s", out)
	}
	if !contains(out, "Your edits were applied.") {
		t.Errorf("message text should appear; got:\n%s", out)
	}

	// role must still be status.
	if !contains(out, `role="status"`) {
		t.Error("success alert missing role=\"status\"")
	}
}
