package server_test

import (
	"os"
	"strings"
	"testing"
)

// A page left open keeps its days true (36-days.js): it follows itself
// through the same refresh a change uses, at the workspace's midnight and
// when a time it shows passes, and waits while hidden. The zone said when
// the reader's differs is checked in a browser: behave-page.mjs.
func TestAPageLeftOpenKeepsItsDaysTrue(t *testing.T) {
	data, err := os.ReadFile("../../design/base/36-days.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(data)
	for _, want := range []string{
		`getAttribute("data-zone")`, `getAttribute("data-today")`, // the server's zone and day
		"sw.refresh(0)",                // the refresh that keeps focus and scroll
		`time[datetime*='T']`,          // a time shown passing
		`visibilityState === "hidden"`, // a hidden tab waits
		`"visibilitychange"`,           // and catches up when seen
	} {
		if !strings.Contains(js, want) {
			t.Errorf("36-days.js does not have %s", want)
		}
	}
}
