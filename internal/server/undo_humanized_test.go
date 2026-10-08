package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Undo of an already-humanized setting change reads in plain words on /activity:
// "You undid: Assistant changed spacing to Wide", not
// "You undid:: Assistant changed Room between lines and words to Wide".
// This covers acceptance items 1–3 of task 0227.
func TestUndoHumanizedSettingReadsInPlainWords(t *testing.T) {
	a, h := newApp(t)

	// Seed an original setting-change entry with already-humanized text
	// (as it would be stored if the assistant previously wrote it in words).
	old, err := a.Store.Create(records.ActivityType, map[string]any{
		"summary": "Assistant changed Room between lines and words to Wide",
		"actor":   "assistant",
		"action":  "set",
		"target":  "ui.spacing",
		"detail":  "wide",
	})
	if err != nil {
		t.Fatal(err)
	}

	// The undo entry carries the raw summary from the original change.
	if _, err := a.Store.Create(records.ActivityType, map[string]any{
		"summary": "You undid: Assistant changed Room between lines and words to Wide",
		"actor":   "human",
		"action":  "set",
		"target":  "ui.spacing",
		"detail":  "normal",
		"undoes":  old.ID,
	}); err != nil {
		t.Fatal(err)
	}

	page := get(t, h, "/activity").Body.String()
	text := said(page)

	// Acceptance 1: no internal field names like "Room between lines".
	if strings.Contains(text, "Room between lines") {
		t.Errorf("the log must not show the raw description 'Room between lines'; it should say 'spacing'\n\npage text:\n%s", page)
	}

	// Acceptance 1: no dotted internal keys.
	if strings.Contains(page, `data-target="ui.spacing"`) {
		t.Errorf("the log must not show the internal key 'ui.spacing'\n\npage:\n%s", page)
	}

	// Acceptance 3: single colon only — "You undid:" with one colon.
	if strings.Contains(page, "undid::") || strings.Contains(text, "undid::") {
		t.Errorf("the log must not have a doubled colon; it should be 'You undid:' with one colon\n\npage:\n%s", page)
	}

	// Acceptance 1 & 2: the entry reads in plain language.
	if !strings.Contains(text, "Assistant changed spacing to Wide") {
		t.Errorf("the undo summary must read in plain words like 'Assistant changed spacing to Wide'\n\npage text:\n%s", page)
	}
}
