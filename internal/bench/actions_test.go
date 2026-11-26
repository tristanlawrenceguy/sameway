package bench

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
)

// actions are the ways a person gets something done for them: a button
// that calls an address, one that runs at a time, one that reads a page
// and tells them, and one the assistant is asked to press, which reaches
// outside and so is put to the person first. (One that runs when a record
// changes is "automate" in requests_test.go.)
func actions() []request {
	return []request{
		{name: "act-webhook", say: "Make me a button called Alarm on that posts to https://example.com/alarm/on",
			check: func(a *app.App, _ string) string {
				act := find(a, "action", "url", "example.com/alarm/on")
				return want(act != nil && (act.Fields["kind"] == "webhook" || act.Fields["kind"] == nil), "no webhook action for the address")
			}},
		{name: "act-scheduled", say: "Every day at 7:30 get https://example.com/weather and keep the answer on my page.",
			check: func(a *app.App, _ string) string {
				act := find(a, "action", "url", "example.com/weather")
				if act == nil {
					return "no action for the address"
				}
				return all(want(act.Fields["every"] == "day" && strings.Contains(fmt.Sprint(act.Fields["at"]), "7:30"), fmt.Sprint("it runs every ", act.Fields["every"], " at ", act.Fields["at"])),
					want(act.Fields["show"] == true, "its answer is not shown"))
			}},
		{name: "act-watch", say: "Keep an eye on https://example.com/boots and tell me when the price is under 80.",
			check: func(a *app.App, _ string) string {
				act := find(a, "action", "url", "example.com/boots")
				if act == nil {
					return "no action for the page"
				}
				every := fmt.Sprint(act.Fields["every"])
				return all(want(act.Fields["kind"] == "watch" && strings.Contains(fmt.Sprint(act.Fields["watch_for"]), "80"), fmt.Sprint("it is a ", act.Fields["kind"], " waiting for ", act.Fields["watch_for"])),
					want(every == "hour" || every == "day" || every == "week", "it reads the page every "+every))
			}},
		{name: "act-run", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "action", map[string]any{"title": "Open the gate", "kind": "webhook", "url": "https://example.invalid/gate/open"})
		}, say: "Open the gate.",
			check: func(a *app.App, _ string) string {
				return want(proposes(a, "run_action"), "pressing it was not put to the person")
			}},
	}
}
