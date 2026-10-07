package bench

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// newcomer are the first things a person asks in their first ten minutes,
// the page's own suggestions among them, on a new workspace and with the
// model most of them have: the free one on their own computer. They are
// easy for Claude and were not for Qwen 3.5, which put a weekend a week out
// and answered what it could do with a list of record kinds.
//
//	SAMEWAY_BENCH_SET=newcomer SAMEWAY_BENCH_MODEL=sameway-qwen3.5 SAMEWAY_BENCH_BASE_URL=http://127.0.0.1:11434/v1 go test ./internal/bench -run Assistant -v
func newcomer() []request {
	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	return []request{
		{name: "first-task", say: "Add a task for tomorrow: call the dentist",
			check: func(a *app.App, _ string) string {
				r := find(a, "task", "title", "dentist")
				return all(want(r != nil, "no dentist task"), want(r == nil || onDay(r, tomorrow), "it is not due tomorrow"))
			}},
		{name: "first-habit", say: "Keep a habit: 8 glasses of water a day",
			check: func(a *app.App, _ string) string {
				h := find(a, "habit", "name", "water")
				if h == nil {
					h = find(a, "habit", "name", "glass")
				}
				return all(want(h != nil, "no water habit"),
					want(h == nil || num(h.Fields["target"]) == 8 && has(h, "cadence", "day"), fmt.Sprintf("not 8 a day: %v", h)),
					block(a, "tracker", ""))
			}},
		{name: "first-list", say: "Make a list of what I need to buy: milk, bread and eggs",
			check: func(a *app.App, _ string) string {
				var all []string
				for _, typ := range []string{"note", "task"} {
					for _, r := range list(a, typ) {
						all = append(all, strings.ToLower(jsonOf(r.Fields)))
					}
				}
				for _, b := range list(a, chat.BlockType) {
					all = append(all, strings.ToLower(jsonOf(b.Fields["props"])))
				}
				got := strings.Join(all, " ")
				return want(strings.Contains(got, "milk") && strings.Contains(got, "bread") && strings.Contains(got, "eggs"), "milk, bread and eggs are not all kept")
			}},
		{name: "weekend", say: "Plan my weekend: a walk on Saturday morning and call grandma on Sunday",
			check: func(a *app.App, _ string) string {
				walk, gran := anyTitled(a, "walk"), anyTitled(a, "grandma")
				return all(want(walk != nil && onDay(walk, next(time.Saturday)), "the walk is not on the coming Saturday"),
					want(gran != nil && onDay(gran, next(time.Sunday)), "grandma is not on the coming Sunday"))
			}},
		{name: "tonight", say: "Remind me to take the bins out tonight at 8",
			check: func(a *app.App, _ string) string {
				r := anyTitled(a, "bin")
				return all(want(r != nil, "nothing about the bins"), want(r == nil || onDay(r, today) && (hourIs(r, "at", 20) || hourIs(r, "due", 20)), "it is not today at 20:00"))
			}},
		{name: "tick", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Call the dentist", "due": tomorrow})
			mustCreate(t, a, "task", map[string]any{"title": "Book the car service", "due": tomorrow})
		}, say: "I called the dentist",
			check: func(a *app.App, _ string) string {
				return all(want(has(find(a, "task", "title", "dentist"), "done", "true"), "the dentist task is not done"),
					want(!has(find(a, "task", "title", "car"), "done", "true"), "the car task was ticked too"))
			}},
		{name: "this-week", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Renew passport", "due": tomorrow})
			mustCreate(t, a, "task", map[string]any{"title": "Send the invoice", "due": time.Now().AddDate(0, 0, 3).Format("2006-01-02")})
			mustCreate(t, a, "task", map[string]any{"title": "Plan the holiday", "due": time.Now().AddDate(0, 0, 30).Format("2006-01-02")})
		}, say: "What do I have to do this week?",
			check: func(_ *app.App, reply string) string {
				r := strings.ToLower(reply)
				return all(want(strings.Contains(r, "passport") && strings.Contains(r, "invoice"), "the week's tasks are not in the answer"),
					want(!strings.Contains(r, "holiday"), "next month's task was given as this week's"))
			}},
		{name: "birthday", say: "Mum's birthday is on 3 March, every year",
			check: func(a *app.App, _ string) string {
				r := anyTitled(a, "birthday")
				if r == nil {
					return "no birthday"
				}
				d := fmt.Sprint(r.Fields["starts"], r.Fields["at"], r.Fields["due"])
				return all(want(strings.Contains(d, "-03-03"), "it is not on 3 March: "+d),
					want(strings.Contains(strings.ToUpper(fmt.Sprint(r.Fields["repeat"])), "YEARLY"), "it does not repeat every year"))
			}},
		{name: "show-tasks", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Renew passport", "due": tomorrow})
		}, say: "Show my tasks on the page",
			check: func(a *app.App, _ string) string { return block(a, "collection", "task") }},
		{name: "setup-home", say: "Set Sameway up for my home. I want to keep track of shopping, chores and bills. Don't ask me anything, just set it up.",
			check: func(a *app.App, _ string) string { return setUp(a, "bill") }},
		{name: "setup-work", say: "Set Sameway up for my work. I want to keep track of tasks, meetings and clients. Don't ask me anything, just set it up.",
			check: func(a *app.App, _ string) string { return setUp(a, "client") }},
		{name: "what-can", say: "What can you do here?",
			check: func(_ *app.App, reply string) string {
				r := strings.ToLower(reply)
				return all(want(len(reply) > 40, "no answer"),
					want(!strings.Contains(r, "/t/") && !strings.Contains(r, " records") && !strings.Contains(r, "canvas") && !strings.Contains(r, "content type"), "it answered in machine words: "+reply))
			}},
	}
}

// anyTitled is the first record of any kind titled with the words: a
// model may file a call as an event, a task, a reminder or an interaction.
func anyTitled(a *app.App, words string) *store.Record {
	for _, typ := range a.Store.Types().Names() {
		if r := find(a, typ, "title", words); r != nil {
			return r
		}
	}
	return nil
}

// setUp says whether a set-up was built: something of the kind named (a
// kind of record, or a list showing one), at least two things on the
// page besides the chat, and nothing made up to fill them.
func setUp(a *app.App, kind string) string {
	blocks, kinded := 0, false
	for _, b := range list(a, chat.BlockType) {
		if b.Fields["component"] == "chat" {
			continue
		}
		blocks++
		if strings.Contains(strings.ToLower(jsonOf(b.Fields["props"])), kind) {
			kinded = true
		}
	}
	for _, name := range a.Store.Types().Names() {
		if strings.Contains(name, kind) {
			kinded = true
		}
	}
	made := 0
	for _, t := range a.Types.Types {
		if t.Internal {
			continue
		}
		n, _ := a.Store.Count(t.Name)
		made += n
	}
	return all(want(kinded, "nothing for "+kind+"s"), want(blocks >= 2, fmt.Sprintf("%d things on the page", blocks)), want(made == 0, fmt.Sprintf("%d records made up", made)))
}
