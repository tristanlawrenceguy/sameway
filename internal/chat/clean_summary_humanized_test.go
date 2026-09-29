package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Already-humanized setting change summaries in undo context are cleaned back
// to the same format as old-style ones: "Room between lines and words" maps
// to "spacing", so "You undid: Assistant changed Room between lines and words
// to Wide" becomes "You undid: Assistant changed spacing to Wide". This covers
// acceptance item 2 of task 0227.
func TestCleanSummaryHumanizedSettingUndo(t *testing.T) {
	for in, want := range map[string]string{
		"You undid: Assistant changed Room between lines and words to Wide":           "You undid: Assistant changed spacing to Wide",
		"You undid: Assistant changed How large the words are to Large":               "You undid: Assistant changed text size to Large",
		"Bob put back: Assistant changed Pace to Calm":                                "Bob put back: Assistant changed pace to Calm",
		"You undid: Assistant changed Text size to Normal":                            "You undid: Assistant changed text size to Normal",
		"You undid: Assistant changed Room between lines and words to Wide, on phone": "You undid: Assistant changed spacing to Wide, on phone",
		// Label-as-prefix: humanized summary prepends the label before the description fragment.
		"You undid: Assistant changed Pace — how changes arrive to Calmly": "You undid: Assistant changed pace to Calmly",
	} {
		if got := chat.CleanSummary(in); got != want {
			t.Errorf("CleanSummary(%q) = %q, want %q", in, got, want)
		}
	}
}
