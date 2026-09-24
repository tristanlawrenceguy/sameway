package server_test

// Tests that link component examples with look="button" have labels ≤3 words.
// These pin Acceptance 1 (≤3 words) for all link buttons, including the
// "Back to the canvas" example used on focus pages.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/design"
)

// TestLinkButtonExamplesLabelsAreShort reads the link component manifest and checks
// every example with look="button" has a label of ≤3 words.  This covers Acceptance 1
// for all system-level link-button examples, including "Back to the canvas".
func TestLinkButtonExamplesLabelsAreShort(t *testing.T) {
	data, err := design.FS.ReadFile("components/link/manifest.json")
	if err != nil {
		t.Fatalf("cannot read link manifest: %v", err)
	}

	var manifest struct {
		Examples []struct {
			Name  string          `json:"name"`
			Props json.RawMessage `json:"props"`
		} `json:"examples"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("cannot parse link manifest: %v", err)
	}

	for _, ex := range manifest.Examples {
		var props map[string]any
		if err := json.Unmarshal(ex.Props, &props); err != nil {
			t.Fatalf("%s: cannot parse example props: %v", ex.Name, err)
		}
		look, _ := props["look"].(string)
		if look != "button" {
			continue // only check link buttons
		}
		label, _ := props["label"].(string)
		n := len(strings.Fields(label))
		if n > 3 {
			t.Errorf("link button example %q label %q is %d words — must be ≤3 (Acceptance 1)", ex.Name, label, n)
		}
	}
}
