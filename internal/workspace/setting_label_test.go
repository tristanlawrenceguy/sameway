package workspace_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// TestSettingLabelPace checks that "ui.pace" returns "Pace".
func TestSettingLabelPace(t *testing.T) {
	t.Parallel()
	got := workspace.SettingLabel("ui.pace")
	want := "Pace"
	if got != want {
		t.Errorf("ui.pace should be %q, got %q", want, got)
	}
}

// TestSettingLabelSpacing checks that "ui.spacing" returns "Spacing".
func TestSettingLabelSpacing(t *testing.T) {
	t.Parallel()
	got := workspace.SettingLabel("ui.spacing")
	want := "Spacing"
	if got != want {
		t.Errorf("ui.spacing should be %q, got %q", want, got)
	}
}
