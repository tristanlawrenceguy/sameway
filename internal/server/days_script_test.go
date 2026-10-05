package server_test

import (
	"os"
	"strings"
	"testing"
)

// A page left open keeps its days true (30-days.js): it follows itself
// through the same refresh a change uses, at the workspace's midnight and
// when a time it shows passes, waits while hidden, and says the
// workspace's zone, again after a refresh, when the reader's differs.
func TestAPageLeftOpenKeepsItsDaysTrue(t *testing.T) {
	data, err := os.ReadFile("../../design/base/30-days.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(data)
	for _, want := range []string{
		`getAttribute("data-zone")`, `getAttribute("data-today")`, // the server's zone and day
		"window.swRefresh(0)",                 // the refresh that keeps focus and scroll
		`time[datetime*='T']`,                 // a time shown passing
		`visibilityState === "hidden"`,        // a hidden tab waits
		`"visibilitychange"`,                  // and catches up when seen
		"getTimezoneOffset",                   // the reader's zone
		"Times here are this workspace's",     // said in words
		`addEventListener("sw:refresh", say)`, // and again after a refresh
	} {
		if !strings.Contains(js, want) {
			t.Errorf("30-days.js does not have %s", want)
		}
	}
}
