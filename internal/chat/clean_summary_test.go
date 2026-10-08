package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Summaries stored in keys before setting names existed are said in words,
// inside an undo too; every other summary is left as it was.
func TestCleanSummary(t *testing.T) {
	for in, want := range map[string]string{
		"You set ui.text normal":                   "You changed text size to Normal",
		"Assistant set ui.spacing wide":            "Assistant changed spacing to Wide",
		"You undid: You set ui.text large":         "You undid: You changed text size to Large",
		"Bob put back: Assistant set ui.pace calm": "Bob put back: Assistant changed pace to Calm",
		"You set ui.controls visible, on phone":    "You changed item buttons to Visible, on phone",
		"Assistant added card Plan":                "Assistant added card Plan",
		"You undid: Assistant added card Plan":     "You undid: Assistant added card Plan",
	} {
		if got := records.CleanSummary(in); got != want {
			t.Errorf("CleanSummary(%q) = %q, want %q", in, got, want)
		}
	}
}
