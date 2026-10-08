package bench

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// request is one thing a person asks, what is there before, and how to
// tell it was done: check says what is wrong, or nothing.
type request struct {
	name  string
	seed  func(t *testing.T, a *app.App)
	say   string
	check func(a *app.App, reply string) string
}

// requests are the hard ones: several steps at once, changes to what is
// there without touching the rest, days counted, an ambiguity to ask
// about, a layout, an answer from two kinds of record. The easy ones (a
// note, a list, a chart) passed every run once Claude Code was given
// only Read, and were retired.
func requests() []request {
	return []request{
		{name: "repeat", say: "Every Monday at 9am remind me to send my timesheet",
			check: func(a *app.App, _ string) string {
				for _, typ := range []string{"reminder", "task"} {
					if r := find(a, typ, "title", "timesheet"); r != nil {
						rep := strings.ToUpper(fmt.Sprint(r.Fields["repeat"]))
						// Weekly from a Monday is every Monday, BYDAY or not.
						monday := onDay(r, next(time.Monday)) || strings.Contains(rep, "MO")
						return all(want(strings.Contains(rep, "WEEKLY") && monday, "it does not repeat on Mondays: "+rep),
							want(timeIs(r, "at", 9, 0) || timeIs(r, "due", 9, 0), "it is not at 9:00"))
					}
				}
				return "no timesheet reminder"
			}},
		{name: "move-day", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Call the bank", "due": day(time.Thursday, 0)})
			mustCreate(t, a, "task", map[string]any{"title": "Pay rent", "due": day(time.Thursday, 0)})
			mustCreate(t, a, "event", map[string]any{"title": "Team lunch", "starts": at(day(time.Thursday, 0), 12, 30)})
			mustCreate(t, a, "task", map[string]any{"title": "Water plants", "due": day(time.Wednesday, 0)})
		}, say: "Move everything I have on Thursday to Friday",
			check: func(a *app.App, _ string) string {
				fri := day(time.Thursday, 1)
				bank, rent, lunch, water := find(a, "task", "title", "bank"), find(a, "task", "title", "rent"), find(a, "event", "title", "lunch"), find(a, "task", "title", "water")
				return all(want(onDay(bank, fri) && onDay(rent, fri), "the tasks are not on Friday"),
					want(onDay(lunch, fri) && timeIs(lunch, "starts", 12, 30), fmt.Sprintf("lunch is at %v, not Friday 12:30", lunch.Fields["starts"])),
					want(onDay(water, day(time.Wednesday, 0)), "Wednesday's task was moved"))
			}},
		{name: "bulk-tag", seed: func(t *testing.T, a *app.App) {
			for _, n := range [][2]string{{"Tomato seedlings", "Pot them on in May."}, {"Compost bins", "Turn the heap monthly."}, {"Pruning roses", "Cut back in February."},
				{"Tax return", "Due in January."}, {"Book club", "Next: Middlemarch."}, {"Car service", "Booked for Tuesday."}} {
				mustCreate(t, a, "note", map[string]any{"title": n[0], "body": n[1]})
			}
		}, say: "Tag my garden notes with garden",
			check: func(a *app.App, _ string) string {
				var wrong []string
				for _, n := range list(a, "note") {
					title := fmt.Sprint(n.Fields["title"])
					garden := strings.Contains("Tomato seedlings|Compost bins|Pruning roses", title)
					if tagged(n, "garden") != garden {
						wrong = append(wrong, title)
					}
				}
				return want(len(wrong) == 0, "tagged wrong: "+strings.Join(wrong, ", "))
			}},
		{name: "follow-up", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "person", map[string]any{"name": "Ana Silva"})
			mustCreate(t, a, "person", map[string]any{"name": "Joe Brown"})
			mustCreate(t, a, "event", map[string]any{"title": "Pricing review", "starts": at(time.Now().AddDate(0, 0, -2).Format("2006-01-02"), 10, 0),
				"notes": "Ana Silva will send the new price list by 9 October. Joe Brown will update the website once it is out. We agreed to raise prices by 5%."})
		}, say: "Turn my notes from the pricing review into tasks for who said they'd do what",
			check: func(a *app.App, _ string) string {
				ana, joe := find(a, "person", "name", "ana"), find(a, "person", "name", "joe")
				list, site := find(a, "task", "title", "price list"), find(a, "task", "title", "website")
				if list == nil || site == nil {
					return "no tasks for the price list and the website"
				}
				return all(want(fmt.Sprint(list.Fields["for"]) == ana.ID, "the price list is not Ana's"), want(fmt.Sprint(site.Fields["for"]) == joe.ID, "the website is not Joe's"),
					want(onDay(list, fmt.Sprintf("%d-10-09", time.Now().Year())), fmt.Sprintf("the price list is due %v, not 9 October", list.Fields["due"])))
			}},
		{name: "today-tab", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Post the parcel", "due": time.Now().Format("2006-01-02")})
			mustCreate(t, a, "event", map[string]any{"title": "Physio", "starts": at(time.Now().Format("2006-01-02"), 15, 0)})
			mustCreate(t, a, "habit", map[string]any{"name": "Drink water", "cadence": "day", "target": 8, "unit": "glasses"})
		}, say: "Make me a Today tab with what's due today, today's events and my water habit",
			check: func(a *app.App, _ string) string {
				c := find(a, records.CanvasType, "name", "today")
				if c == nil {
					return "no Today tab"
				}
				return all(want(on(a, c.ID, "task"), "no tasks on it"), want(on(a, c.ID, "event"), "no events on it"), want(on(a, c.ID, "habit") || on(a, c.ID, "tracker"), "no habit on it"))
			}},
		{name: "ambiguous", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Send invoice to Ana", "status": "todo"})
			mustCreate(t, a, "task", map[string]any{"title": "Send invoice to Joe", "status": "todo"})
		}, say: "Mark the invoice task done",
			check: func(a *app.App, reply string) string {
				for _, r := range list(a, "task") {
					if r.Fields["status"] == "done" || r.Fields["done"] == true {
						return "it guessed: " + fmt.Sprint(r.Fields["title"]) + " was ticked"
					}
				}
				return want(strings.Contains(reply, "?"), "it neither asked nor did anything")
			}},
		{name: "next-week", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "event", map[string]any{"title": "Dentist", "starts": at(day(time.Tuesday, 0), 10, 0)})
		}, say: "My dentist appointment moved to the same time a week later",
			check: func(a *app.App, _ string) string {
				d := find(a, "event", "title", "dentist")
				return want(onDay(d, day(time.Tuesday, 7)) && timeIs(d, "starts", 10, 0), fmt.Sprintf("it starts %v, not %s 10:00", d.Fields["starts"], day(time.Tuesday, 7)))
			}},
		{name: "priority", seed: func(t *testing.T, a *app.App) {
			for _, title := range []string{"Fix the leak [urgent]", "Book flights", "Renew insurance [urgent]", "Clean the garage"} {
				mustCreate(t, a, "task", map[string]any{"title": title})
			}
		}, say: "Give tasks a priority (low, medium or high) and set the urgent ones to high",
			check: func(a *app.App, _ string) string {
				var wrong []string
				for _, r := range list(a, "task") {
					title := fmt.Sprint(r.Fields["title"])
					if (r.Fields["priority"] == "high") != strings.Contains(title, "urgent") {
						wrong = append(wrong, title)
					}
				}
				return want(len(wrong) == 0, "priority wrong on: "+strings.Join(wrong, ", "))
			}},
		{name: "split", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "note", map[string]any{"title": "Trip report", "body": "## Day 1\nWe arrived in Porto.\n\n## Day 2\nWe walked to the river.\n\n## Day 3\nWe flew home."})
		}, say: "Split my Trip report into one note per day, kept as its parts in order",
			check: func(a *app.App, _ string) string {
				trip := find(a, "note", "title", "trip report")
				var found []string
				for _, n := range list(a, "note") {
					if fmt.Sprint(n.Fields[records.PartOf]) == trip.ID {
						found = append(found, fmt.Sprint(n.Fields["body"]))
					}
				}
				words := strings.Join(found, " ")
				return want(len(found) == 3 && strings.Contains(words, "Porto") && strings.Contains(words, "river") && strings.Contains(words, "flew"), fmt.Sprintf("%d parts", len(found)))
			}},
		{name: "automate", say: "When a task tagged client is marked done, send its title and who it was for to https://example.com/hook",
			check: func(a *app.App, _ string) string {
				act := find(a, "action", "url", "example.com")
				if act == nil {
					return want(len(list(a, "proposal")) > 0, "no action and nothing asked")
				}
				only, sent := strings.ToLower(jsonOf(act.Fields["only"])), fmt.Sprint(act.Fields["url"], act.Fields["body"], act.Fields["payload"])
				return all(want(act.Fields["what"] == "task" && strings.Contains(only, "client") && strings.Contains(only, "done"), "it runs on "+fmt.Sprint(act.Fields["what"], " ", only)),
					want(strings.Contains(sent, "{{title}}") && strings.Contains(sent, "{{for}}"), "it does not send the title and who it was for: "+sent))
			}},
		{name: "two-kinds", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "note", map[string]any{"title": "Passport", "body": "Renewal costs £88.50 online, about three weeks."})
			mustCreate(t, a, "task", map[string]any{"title": "Renew passport", "due": fmt.Sprintf("%d-11-12", time.Now().Year())})
		}, say: "When do I need to renew my passport, and what will it cost?",
			check: func(_ *app.App, reply string) string {
				return all(want(strings.Contains(reply, "88.50"), "the cost is not said"), want(strings.Contains(reply, "12"), "the day is not said"))
			}},
		{name: "people", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "event", map[string]any{"title": "Pricing", "starts": at(day(time.Tuesday, 0), 14, 0)})
		}, say: "Add Ana Silva (ana@example.com) and Joe Brown (joe@example.com) and put them both on the Pricing meeting",
			check: func(a *app.App, _ string) string {
				ana, joe, ev := find(a, "person", "email", "ana@"), find(a, "person", "email", "joe@"), find(a, "event", "title", "pricing")
				if ana == nil || joe == nil {
					return "Ana or Joe is missing, or without an email"
				}
				people := jsonOf(ev.Fields["people"])
				return want(strings.Contains(people, ana.ID) && strings.Contains(people, joe.ID), "the meeting's people are "+people)
			}},
		{name: "layout", seed: func(t *testing.T, a *app.App) {
			blockOf(t, a, "collection", map[string]any{"type": "task", "label": "Shopping list"})
			blockOf(t, a, "collection", map[string]any{"type": "note", "label": "Notes"})
			blockOf(t, a, "calendar", map[string]any{"type": "event", "caption": "Calendar"})
		}, say: "Put my shopping list and my notes side by side, with the calendar full width under them",
			check: func(a *app.App, _ string) string {
				var shop, notes, cal map[string]any
				for _, b := range list(a, records.BlockType) {
					switch p := jsonOf(b.Fields["props"]); {
					case strings.Contains(p, "Shopping"):
						shop = b.Fields
					case strings.Contains(p, `"Notes"`):
						notes = b.Fields
					case b.Fields["component"] == "calendar":
						cal = b.Fields
					}
				}
				if shop == nil || notes == nil || cal == nil {
					return "a block went missing"
				}
				return all(want(num(shop["span"]) == 6 && num(notes["span"]) == 6, fmt.Sprintf("spans %v and %v, not 6 and 6", shop["span"], notes["span"])),
					want(num(cal["span"]) == 12 && num(cal["position"]) > num(shop["position"]) && num(cal["position"]) > num(notes["position"]), "the calendar is not full width under them"))
			}},
	}
}
