package server_test

import (
	"os"
	"strings"
	"testing"
)

// A model on this computer can think for most of a minute before its first
// word; the turn's Thinking says, after a while, that this is how it goes,
// so the page does not read as stuck (14-live.js).
func TestALongThinkSaysItIsStillGoing(t *testing.T) {
	data, err := os.ReadFile("../../design/base/14-live.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(data)
	for _, want := range []string{"Still thinking. Some models take a minute.", "setTimeout(", "clearTimeout(slow)"} {
		if !strings.Contains(js, want) {
			t.Errorf("14-live.js does not have %s", want)
		}
	}
}
