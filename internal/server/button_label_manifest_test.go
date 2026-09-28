package server_test

// Tests that button component examples have labels ≤3 words.
// These pin Acceptance 1 (≤3 words) across all built-in button examples.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
)

// TestButtonExamplesLabelsAreShort reads the button component manifest and checks
// every example's label is ≤3 words.  This covers Acceptance 1 for all system-level
// button examples, including the action example "Turn on the alarm".
func TestButtonExamplesLabelsAreShort(t *testing.T) {
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
		var props map[string]any
		if err := json.Unmarshal(ex.Props, &props); err != nil {
			t.Fatalf("%s: cannot parse example props: %v", ex.Name, err)
		}
		label, _ := props["label"].(string)
		n := len(strings.Fields(label))
		if n > 3 {
			t.Errorf("button example %q label %q is %d words — must be ≤3 (Acceptance 1)", ex.Name, label, n)
		}
	}
}
