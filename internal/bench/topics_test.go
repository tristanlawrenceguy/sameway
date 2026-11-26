package bench

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// topics are requests that each lean on one topic's rules
// (chat/rules_topical.go): a chart from records, a calendar of a habit's
// days, an allowance, a reminder about a thing, a button that runs on its
// own, a file's words, a setting, a need.
func topics() []request {
	return []request{
		{name: "topic-chart", seed: func(t *testing.T, a *app.App) {
			for title, status := range map[string]string{"Fix the leak": "todo", "Book flights": "doing", "Renew insurance": "done", "Clean the garage": "done"} {
				mustCreate(t, a, "task", map[string]any{"title": title, "status": status})
			}
		}, say: "Show me a chart of how many tasks I have in each status",
			check: func(a *app.App, _ string) string {
				p := propsOf(a, "chart")
				return want(p != nil && p["type"] == "task" && p["by"] == "status", "no chart counting tasks by status: "+jsonOf(p))
			}},
		{name: "topic-calendar", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "habit", map[string]any{"name": "Stretch", "cadence": "day", "target": 1})
			mustCreate(t, a, "entry", map[string]any{"habit": find(a, "habit", "name", "stretch").ID, "at": time.Now().AddDate(0, 0, -1).Format("2006-01-02"), "amount": 1})
		}, say: "Put a calendar on the page that shows the days I did my Stretch habit",
			check: func(a *app.App, _ string) string {
				p := propsOf(a, "calendar")
				return all(want(p != nil && p["type"] == "entry" && strings.Contains(jsonOf(p["where"]), find(a, "habit", "name", "stretch").ID), "no calendar of the habit's entries: "+jsonOf(p)),
					want(p == nil || p["events"] == nil, "the days were copied in as events"))
			}},
		{name: "topic-allowance", say: "I may work at most 21 hours a month. Keep track of it, and log 3 hours for today.",
			check: func(a *app.App, _ string) string {
				var h, e = find(a, "habit", "aim", "limit"), list(a, "entry")
				return all(want(h != nil && has(h, "cadence", "month") && num(h.Fields["target"]) == 21, fmt.Sprintf("no limit of 21 a month: %v", list(a, "habit"))),
					want(len(e) == 1 && num(e[0].Fields["amount"]) == 3, fmt.Sprintf("%d entries, not one of 3 hours", len(e))),
					block(a, "tracker", ""))
			}},
		{name: "topic-remind", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Renew passport"})
		}, say: "Remind me about my Renew passport task on Friday at 9am",
			check: func(a *app.App, _ string) string {
				r, task := find(a, "reminder", "title", "passport"), find(a, "task", "title", "passport")
				return all(want(r != nil && has(r, "about", "/t/task/"+strings.ToLower(task.ID)), fmt.Sprintf("no reminder about the task: %v", r)),
					want(r == nil || onDay(r, next(time.Friday)) && timeIs(r, "at", 9, 0), "it is not on Friday at 9:00"))
			}},
		{name: "topic-button", say: "Make a button on the page called Morning alarm that calls https://example.com/alarm, and have it run by itself every day at 7am too",
			check: func(a *app.App, _ string) string {
				act := find(a, "action", "url", "example.com/alarm")
				if act == nil {
					return "no action that calls the address"
				}
				p := propsOf(a, "button")
				return all(want(has(act, "every", "day") && strings.Contains(fmt.Sprint(act.Fields["at"]), "7"), fmt.Sprintf("it does not run every day at 7: %v %v", act.Fields["every"], act.Fields["at"])),
					want(p != nil && p["action"] == act.ID, "no button for it: "+jsonOf(p)))
			}},
		{name: "topic-file", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "file", map[string]any{"title": "Boiler manual", "name": "boiler.pdf", "kind": "pdf",
				"text": "# Boiler manual\n\n## Pressure\n\nWhen cold, the gauge should read 1.3 bar. Top up with the grey tap under the casing if it is lower.\n\n## Service\n\nOnce a year."})
		}, say: "What does my boiler manual say the pressure should be?",
			check: func(_ *app.App, reply string) string {
				return want(strings.Contains(reply, "1.3"), "the pressure is not said: "+reply)
			}},
		{name: "topic-setting", say: "Things moving on the page bother me. Make changes appear with no motion at all.",
			check: func(a *app.App, _ string) string {
				pace, _, _ := uiSettings(a)
				return want(pace == "still", "changes arrive "+pace)
			}},
		{name: "topic-needs", say: "I use a screen reader, and my partner who shares this needs bigger text.",
			check: func(a *app.App, _ string) string {
				_, text, needs := uiSettings(a)
				return all(want(strings.Contains(strings.ToLower(needs), "screen reader"), "the need was not kept: "+needs),
					want(text == "large" || text == "larger", "the text is "+text))
			}},
	}
}

// propsOf is the props of the first block of a component, or nil.
func propsOf(a *app.App, component string) map[string]any {
	for _, b := range list(a, records.BlockType) {
		if b.Fields["component"] == component {
			var p map[string]any
			json.Unmarshal([]byte(jsonOf(b.Fields["props"])), &p)
			return p
		}
	}
	return nil
}

// uiSettings is how changes arrive, the text size and the needs as the
// workspace file has them now: a setting is changed in Claude Code's
// tool process.
func uiSettings(a *app.App) (pace, text, needs string) {
	w, err := workspace.Load(a.Workspace.Dir)
	if err != nil {
		return "", "", ""
	}
	return w.Config.UI.Pace, w.Config.UI.Text, w.Config.UI.Needs
}
