package bench

import (
	"fmt"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// blocks are changes to what is on a page beside laying it out (layout),
// filling a new tab (today-tab) and taking a block away (remove-block,
// tidy): a block that is there changed, and a whole tab removed.
func blocks() []request {
	return []request{
		{name: "update-block", seed: func(t *testing.T, a *app.App) {
			blockOf(t, a, "collection", map[string]any{"type": "task", "label": "To do"})
			blockOf(t, a, "calendar", map[string]any{"type": "event", "caption": "Calendar"})
		}, say: "Call my To do list Chores instead, and make it full width.",
			check: func(a *app.App, _ string) string {
				n, span := 0, 0
				for _, b := range list(a, records.BlockType) {
					if b.Fields["component"] == "collection" {
						n++
						if has(b, "props", "chores") {
							span = num(b.Fields["span"])
						}
					}
				}
				return all(want(n == 1, fmt.Sprintf("%d lists, not the one renamed", n)), want(span == 12, fmt.Sprintf("the Chores list has span %d, not 12", span)),
					want(block(a, "calendar", "Calendar") == "", "the calendar went"))
			}},
		{name: "remove-tab", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, records.CanvasType, map[string]any{"name": "Garden", "position": 1})
			g := find(a, records.CanvasType, "name", "garden")
			mustCreate(t, a, records.BlockType, a.Chat.BlockFields(map[string]any{"component": "collection", "props": map[string]any{"type": "note", "label": "Garden notes"}, "canvas": g.ID}))
			blockOf(t, a, "collection", map[string]any{"type": "task", "label": "To do"})
		}, say: "I don't need the Garden tab any more. Remove it.",
			check: func(a *app.App, _ string) string {
				return all(want(find(a, records.CanvasType, "name", "garden") == nil || proposes(a, "remove_canvas"), "the Garden tab is still there and nothing asks to remove it"),
					want(block(a, "collection", "To do") == "", "Home's list went"))
			}},
	}
}
