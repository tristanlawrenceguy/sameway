package server_test

import (
	"os"
	"strings"
	"testing"
)

// The tracker amount input label uses only the unit text, never raw schema
// field names wrapped in "Amount in … for {name}". Acceptance 0607: no raw
// schema field names visible on the habits log form.
func TestTrackerLabelHasNoRawSchemaWrappers(t *testing.T) {
	tpl, err := os.ReadFile("../../design/components/tracker/template.html")
	if err != nil {
		t.Fatal(err)
	}

	for _, bad := range []string{
		`<span class="sw-visually-hidden">Amount in </span>`,
		`<span class="sw-visually-hidden"> for {{.shortName}}</span>`,
	} {
		if strings.Contains(string(tpl), bad) {
			t.Errorf("tracker template still has raw schema wrapper %q; label should show only the unit", bad)
		}
	}
}

// The tracker golden example files never show raw schema field name wrappers
// on amount labels; only the unit text appears. Acceptance 0607: no raw
// schema field names visible on the habits log form.
func TestTrackerGoldenHasNoRawSchemaWrappers(t *testing.T) {
	for _, ex := range []string{
		"../../design/components/tracker/examples/default.html",
		"../../design/components/tracker/examples/limit.html",
		"../../design/components/tracker/examples/record.html",
	} {
		data, err := os.ReadFile(ex)
		if err != nil {
			t.Fatalf("%s: %v", ex, err)
		}
		content := string(data)
		for _, bad := range []string{
			`<span class="sw-visually-hidden">Amount in </span>`,
			`<span class="sw-visually-hidden"> for `,
		} {
			if strings.Contains(content, bad) {
				t.Errorf("%s: still has raw schema wrapper %q", ex, bad)
			}
		}
	}
}
