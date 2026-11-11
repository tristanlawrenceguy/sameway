package automate

import (
	"strings"
	"testing"
)

// A long record keeps its start and its end for the model: what a
// conversation decided last is not cut away.
func TestALongRecordKeepsItsEnds(t *testing.T) {
	t.Parallel()
	long := "Plan the move." + strings.Repeat(" words", 2000) + " Book the van for the 14th."
	got := clipEnds(long, 400)
	if !strings.HasPrefix(got, "Plan the move.") || !strings.HasSuffix(got, "Book the van for the 14th.") || len([]rune(got)) > 410 {
		t.Errorf("both ends, about 400: %d %q…", len([]rune(got)), got[:40])
	}
	if clipEnds("short", 400) != "short" {
		t.Error("a short one is as it is")
	}
}
