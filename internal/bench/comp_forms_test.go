package bench

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// compForms are the components a person presses, built by the assistant:
// a button that runs an action there already, one that goes somewhere
// (a link, since a button goes nowhere), files to take away, and a button
// whose action, a message to the assistant, is made with it. Each is
// checked by the block's props, not by what the reply says.
func compForms() []request {
	return []request{
		{name: "button-action", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "action", map[string]any{"title": "Ring the bell", "kind": "webhook", "url": "https://example.com/bell"})
		}, say: "Put a button on my Home page that runs my Ring the bell action",
			check: func(a *app.App, _ string) string {
				act := find(a, "action", "title", "bell")
				return want(act != nil && pressable(a, "button", "action", act.ID), "no button whose action is Ring the bell: "+pressables(a))
			}},
		{name: "button-link", say: "Add a button to my Home page that opens https://example.com/timesheet",
			check: func(a *app.App, _ string) string {
				return want(pressable(a, "link", "href", "https://example.com/timesheet"), "no link to the timesheet: "+pressables(a))
			}},
		{name: "button-export", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Order compost"})
			mustCreate(t, a, "task", map[string]any{"title": "Plant garlic"})
		}, say: "Add a button to my Home page that downloads my tasks as a spreadsheet",
			check: func(a *app.App, _ string) string {
				for _, b := range list(a, records.BlockType) {
					if p := jsonOf(b.Fields["props"]); (b.Fields["component"] == "export" || b.Fields["component"] == "link") && strings.Contains(p, "/export/task.") {
						return ""
					}
				}
				return "no export or link block that downloads the tasks: " + pressables(a)
			}},
		{name: "button-message", say: "Add a button to my Home page that asks you to list what I have due this week",
			check: func(a *app.App, _ string) string {
				for _, act := range list(a, "action") {
					if act.Fields["kind"] == "message" && pressable(a, "button", "action", act.ID) {
						return want(strings.TrimSpace(fmt.Sprint(act.Fields["message"])) != "", "the action says nothing to the assistant")
					}
				}
				return "no button whose action is a message to the assistant: " + pressables(a)
			}},
	}
}

// pressable is whether a block of the component has the prop at that value.
func pressable(a *app.App, component, prop, value string) bool {
	for _, b := range list(a, records.BlockType) {
		var props map[string]any
		json.Unmarshal([]byte(jsonOf(b.Fields["props"])), &props)
		if b.Fields["component"] == component && fmt.Sprint(props[prop]) == value {
			return true
		}
	}
	return false
}

// pressables is each button, link and export block there is, for a failure.
func pressables(a *app.App) string {
	var out []string
	for _, b := range list(a, records.BlockType) {
		switch b.Fields["component"] {
		case "button", "link", "export":
			out = append(out, fmt.Sprint(b.Fields["component"], " ", jsonOf(b.Fields["props"])))
		}
	}
	return strings.Join(out, "; ")
}
