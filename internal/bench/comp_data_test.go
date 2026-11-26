package bench

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/jsontext"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// dataBlocks are the blocks that show records and numbers, asked for in
// a sentence: the block is there, with the props that make it show what
// was asked.
func dataBlocks() []request {
	tasks := func(t *testing.T, a *app.App) {
		mustCreate(t, a, "task", map[string]any{"title": "Renew passport", "due": time.Now().AddDate(0, 0, 2).Format("2006-01-02"), "status": "doing"})
		mustCreate(t, a, "task", map[string]any{"title": "Plan the holiday", "due": time.Now().AddDate(0, 0, 30).Format("2006-01-02")})
		mustCreate(t, a, "task", map[string]any{"title": "Send the invoice", "done": true, "status": "done"})
	}
	return []request{
		{name: "block-due-week", seed: tasks, say: "Put a list of my tasks due this week on my page",
			check: func(a *app.App, _ string) string {
				return blockWith(a, "collection", `"type":"task"`, `due<`)
			}},
		{name: "block-board", seed: tasks, say: "Show my tasks on my page as a board by status",
			check: func(a *app.App, _ string) string {
				return blockWith(a, "collection", `"type":"task"`, `"as":"board"`)
			}},
		{name: "block-chart", seed: tasks, say: "Add a chart to my page of how many tasks I finished each week",
			check: func(a *app.App, _ string) string {
				return all(blockWith(a, "chart", `"type":"task"`, `"period":"week"`), want(blockWith(a, "chart", `"series"`) != "", "the chart's numbers were written by hand"))
			}},
		{name: "block-calendar", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "event", map[string]any{"title": "Dentist", "starts": at(day(time.Thursday, 0), 10, 0)})
		}, say: "Put a calendar of my events on my page",
			check: func(a *app.App, _ string) string {
				return all(want(blockWith(a, "calendar", `"type":"event"`) == "" || blockWith(a, "calendar", `"type":"all"`) == "" || blockWith(a, "calendar", `"types":[`, `"event"`) == "", "no calendar block of events"),
					want(blockWith(a, "calendar", `"events"`) != "", "the calendar's events were written by hand"))
			}},
		{name: "block-tracker", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "habit", map[string]any{"name": "Water", "cadence": "day", "target": 8, "unit": "glasses"})
			mustCreate(t, a, "habit", map[string]any{"name": "Run", "cadence": "week", "target": 3})
		}, say: "Put my habits on my page so I can log them there",
			check: func(a *app.App, _ string) string { return blockWith(a, "tracker") }},
		{name: "block-table", say: "Put a table on my page comparing three phone plans: Basic is £10 for 5GB, Plus is £15 for 20GB, Max is £25 for unlimited data. Columns: Plan, Price, Data.",
			check: func(a *app.App, _ string) string {
				return blockWith(a, "table", `"columns":["Plan","Price","Data"]`, "Basic", "Plus", "Max", "20")
			}},
	}
}

// blockWith says whether a block of the component is on a canvas with
// every one of the words in its props, spaces aside.
func blockWith(a *app.App, component string, words ...string) string {
next:
	for _, b := range list(a, records.BlockType) {
		if b.Fields["component"] != component {
			continue
		}
		raw, _ := jsontext.Marshal(b.Fields["props"]) // as written: < is not escaped
		props := strings.ReplaceAll(string(raw), " ", "")
		for _, w := range words {
			if !strings.Contains(props, w) {
				continue next
			}
		}
		return ""
	}
	return "no " + component + " block with " + strings.Join(words, " and ")
}
