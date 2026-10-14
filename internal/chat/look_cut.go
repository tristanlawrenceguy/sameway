package chat

import (
	"encoding/json"
	"strings"
)

// A long page cut short for the model kept its first 16,000 characters,
// the outline, and lost what comes last: how the tab reads (measured's
// layout, the line the assistant is told before it acts). The page a model
// looked at to judge its layout came back without it whenever the page was
// long, which is when it mattered most. The layout now stays, and the
// outline is cut to make room for it.

const lookCutSaid = "… (cut short: ask again with only, kind or name for the part you need)"

// lookCut is a look's answer within lookLimit, keeping the tab's layout.
func lookCut(out string) string {
	if len(out) <= lookLimit {
		return out
	}
	var looked struct {
		Measured struct {
			Layout string `json:"layout"`
		} `json:"measured"`
	}
	json.Unmarshal([]byte(out), &looked)
	layout := strings.TrimSpace(looked.Measured.Layout)
	if layout == "" || len(layout) > lookLimit/2 {
		return out[:lookLimit] + lookCutSaid
	}
	keep := lookLimit - len(layout)
	return out[:keep] + lookCutSaid + "\n\nHow the tab reads: " + layout
}
