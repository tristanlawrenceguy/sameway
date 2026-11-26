package bench

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
)

// people are what reaches someone outside the workspace: the assistant
// picks the tool, and what it would send waits as a question for the
// owner. Nothing is sent: each checks the question that waits.
func people() []request {
	return []request{
		{name: "people-email", say: "Email Pat Lee (pat@example.com) to say Thursday's visit has moved to Friday at 10am.",
			check: func(a *app.App, _ string) string {
				return waits(a, "send_email", "pat@example.com", "friday")
			}},
		{name: "people-invite", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "event", map[string]any{"title": "Planning dinner", "starts": at(day(time.Friday, 0), 19, 0), "ends": at(day(time.Friday, 0), 21, 0)})
		}, say: "Send Pat Lee (pat@example.com) a calendar invitation to Friday's planning dinner.",
			check: func(a *app.App, _ string) string {
				d := find(a, "event", "title", "planning dinner")
				return waits(a, "send_email", "pat@example.com", `"invite":"`+d.ID+`"`)
			}},
		{name: "people-both-free", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "person", map[string]any{"name": "Sam Hill", "email": "sam@example.com", "sameway": "sam-home.tail1234.ts.net"})
		}, say: "Ask Sam's Sameway when we are both free on " + weekday(time.Thursday) + " evening, 5pm to 10pm.",
			check: func(a *app.App, _ string) string {
				return waits(a, "ask_someone", day(time.Thursday, 0))
			}},
		{name: "people-ask-page", say: "Ask the plumber's page at https://plumber.example.ts.net whether they can come on Thursday morning. I'm Tris, tris@example.com.",
			check: func(a *app.App, _ string) string {
				return waits(a, "ask_page", "plumber.example.ts.net", "thursday", "tris")
			}},
		{name: "people-free", seed: func(t *testing.T, a *app.App) {
			d := day(time.Thursday, 0)
			mustCreate(t, a, "event", map[string]any{"title": "Workshop", "starts": at(d, 8, 0), "ends": at(d, 13, 0)})
			mustCreate(t, a, "event", map[string]any{"title": "Site visit", "starts": at(d, 15, 0), "ends": at(d, 18, 0)})
		}, say: "When am I free in working hours on " + weekday(time.Thursday) + "?",
			check: func(a *app.App, reply string) string {
				return all(want(saysAny(reply, "1pm", "1 pm", "1:00", "13:00") && saysAny(reply, "3pm", "3 pm", "3:00", "15:00"), "it did not say 1pm to 3pm: "+reply),
					want(len(list(a, "proposal")) == 0, "something was put to the owner"))
			}},
		{name: "people-let-in", say: "Let Carol (carol@example.com) look at this workspace, but not change anything.",
			check: func(a *app.App, _ string) string {
				return all(waits(a, "let_in", "carol@example.com", `"access":"view"`),
					want(find(a, "person", "email", "carol@example.com") == nil || find(a, "person", "email", "carol@example.com").Fields["access"] == nil, "she was let in without a yes"))
			}},
	}
}

// weekday is the coming day of that name with its date, as a person says it.
func weekday(wd time.Weekday) string {
	d, _ := time.Parse("2006-01-02", next(wd))
	return d.Format("Monday 2 January")
}

// waits says whether a question waits for the owner to run that tool,
// with each of the words in what it would run.
func waits(a *app.App, tool string, words ...string) string {
	var seen []string
	for _, p := range a.Chat.Proposals() {
		act, _ := p.Fields["action"].(map[string]any)
		seen = append(seen, fmt.Sprint(act["tool"]))
		if act["tool"] != tool {
			continue
		}
		low := strings.ToLower(jsonOf(act))
		missing := ""
		for _, w := range words {
			if !strings.Contains(low, strings.ToLower(w)) {
				missing = w
			}
		}
		if missing == "" {
			return ""
		}
		return fmt.Sprintf("the %s that waits has no %s: %s", tool, missing, low)
	}
	return fmt.Sprintf("no %s waits for a yes (waiting: %v)", tool, seen)
}
