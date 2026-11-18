package bench

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
)

// oneStep are requests that once took a model two steps with ids or
// links copied between them, now one: every record a condition matches
// changed at once, and an action time sets off.
func oneStep() []request {
	return []request{
		{name: "late-done", seed: func(t *testing.T, a *app.App) {
			late := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
			for _, title := range []string{"Return the library books", "Pay the window cleaner", "Send the photos", "Book the MOT", "Reply to Jo"} {
				mustCreate(t, a, "task", map[string]any{"title": title, "due": late})
			}
			mustCreate(t, a, "task", map[string]any{"title": "Plan the party", "due": time.Now().AddDate(0, 0, 10).Format("2006-01-02")})
		}, say: "I did everything that's overdue. Mark it all done.",
			check: func(a *app.App, _ string) string {
				var wrong []string
				for _, r := range list(a, "task") {
					done := fmt.Sprint(r.Fields["done"]) == "true" || fmt.Sprint(r.Fields["status"]) == "done"
					if done == strings.Contains(fmt.Sprint(r.Fields["title"]), "party") {
						wrong = append(wrong, fmt.Sprint(r.Fields["title"]))
					}
				}
				return want(len(wrong) == 0, "wrong: "+strings.Join(wrong, ", "))
			}},
		{name: "day-before", say: "The day before each event, remind me what to bring.",
			check: func(a *app.App, _ string) string {
				act := find(a, "action", "what", "event")
				if act == nil {
					return want(len(list(a, "proposal")) > 0, "no action and nothing asked")
				}
				only := strings.ToLower(jsonOf(act.Fields["only"]))
				return want(act.Fields["when"] == "changed" && strings.Contains(only, "starts<") && strings.Contains(only, "1d"), fmt.Sprint("it runs when ", act.Fields["when"], " ", only))
			}},
	}
}
