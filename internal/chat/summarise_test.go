package chat_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// TestSummariseRecordUsesWordTrimming checks that Summarise for the "record"
// component uses word-count trimming on the record title, not character-boundary
// truncation. This covers acceptance item 1: canvas block operations involving
// record blocks must also show 6-word trimmed titles in receipts.
func TestSummariseRecordUsesWordTrimming(t *testing.T) {
	const longTitle = "This is a note with a very long multi-word title that exceeds six words and should be trimmed"
	wantTrimmed := "This is a note with a…"

	got := chat.Summarise("record", map[string]any{
		"type":   "note",
		"record": longTitle,
	})

	if !strings.Contains(got, wantTrimmed) {
		t.Errorf("Summarise(record) should use 6-word trimmed title %q\n\ngot: %q", wantTrimmed, got)
	}

	// Make sure it's not character-boundary truncated.
	charTruncated := "This is a note with a very long multi-word title that excee…"
	if strings.Contains(got, charTruncated) {
		t.Errorf("Summarise(record) should not use character-boundary truncation\n\nFound: %q", got)
	}

	// The result must end with an ellipsis when trimmed.
	if !strings.HasSuffix(got, "…") {
		t.Errorf("trimmed record summary should end with ellipsis (U+2026), got: %q", got)
	}
}

// TestSummariseShortRecordNotTrimmed checks that records with six or fewer
// words in the title are not trimmed by Summarise.
func TestSummariseShortRecordNotTrimmed(t *testing.T) {
	got := chat.Summarise("record", map[string]any{
		"type":   "note",
		"record": "Call the dentist",
	})

	want := "note Call the dentist"
	if got != want {
		t.Errorf("Summarise(record) should not trim short titles\n\nwant: %q\ngot:  %q", want, got)
	}
}
