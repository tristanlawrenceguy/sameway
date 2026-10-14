package chat

import (
	"encoding/json"
	"strings"
	"testing"
)

// A look too long for the model is cut in its outline, and the tab's
// layout, which comes last, is kept; a short one is as it was.
func TestALongLookKeepsHowTheTabReads(t *testing.T) {
	t.Parallel()
	raw, _ := json.Marshal(map[string]any{
		"outline":  map[string]any{"text": strings.Repeat("a heading and a control ", 2000)},
		"measured": map[string]any{"layout": "Layout now: row 1: Tasks collection 12. To look at: (1) block x is the same as block y."},
	})
	got := lookCut(string(raw))
	if len(got) > lookLimit+len(lookCutSaid)+len("\n\nHow the tab reads: ") {
		t.Errorf("cut to the limit: %d", len(got))
	}
	if !strings.HasSuffix(got, "How the tab reads: Layout now: row 1: Tasks collection 12. To look at: (1) block x is the same as block y.") || !strings.Contains(got, "cut short") {
		t.Errorf("the layout is kept after the cut outline:\n%s", got[len(got)-300:])
	}
	short := `{"outline":{},"measured":{"layout":"Layout now"}}`
	if lookCut(short) != short {
		t.Error("a short look is as it was")
	}
}
