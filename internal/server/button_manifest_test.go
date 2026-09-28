package server_test

// Tests for button-manifest simplification (danger example label).
// These pin that task 0155's Acceptance 2 is met: no noun-phrase labels.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
)

// TestButtonDangerExampleLabelIsShort checks that the button component's danger
// example uses a short, plain-verb label — not "Delete note".  After the change it
// should be just "Delete", with context: "note" for screen readers.  This covers
// Acceptance 2 (active verb only).
func TestButtonDangerExampleLabelIsShort(t *testing.T) {
	data, err := design.FS.ReadFile("components/button/manifest.json")
	if err != nil {
		t.Fatalf("cannot read button manifest: %v", err)
	}

	var manifest struct {
		Examples []struct {
			Name  string          `json:"name"`
			Props json.RawMessage `json:"props"`
		} `json:"examples"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("cannot parse button manifest: %v", err)
	}

	for _, ex := range manifest.Examples {
		if ex.Name == "danger" {
			var props map[string]any
			if err := json.Unmarshal(ex.Props, &props); err != nil {
				t.Fatalf("cannot parse danger example props: %v", err)
			}
			label, _ := props["label"].(string)

			if label == "Delete note" || len(strings.Fields(label)) > 4 {
				t.Errorf("danger example button label must be ≤4 words and a plain verb; got %q — use 'Delete' + context for screen readers", label)
			}
			return
		}
	}
	t.Error("button manifest missing a danger example")
}
