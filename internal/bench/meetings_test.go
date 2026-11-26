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

const standup = "[0:00] **Ann:** Morning, two things.\n\n[0:12] **Ben:** We ship on Friday, then.\n\n[1:05] **Ann:** Ben, can you email the client?"

// meetings are a meeting written up (from notes, from a recording, onto
// the event it was) and one set to be recorded, by Sameway or by the
// meeting app.
func meetings() []request {
	recording := func(t *testing.T, a *app.App, title string) {
		mustCreate(t, a, "file", map[string]any{"title": title, "kind": "audio", "text": standup})
	}
	review := func(t *testing.T, a *app.App) {
		mustCreate(t, a, "event", map[string]any{"title": "Design review", "starts": at(day(time.Tuesday, 0), 14, 0), "ends": at(day(time.Tuesday, 0), 15, 0)})
	}
	// asksToRecord is whether the meeting has the reminder whose title starts so.
	asksToRecord := func(a *app.App, title string) string {
		ev := find(a, "event", "title", "design review")
		for _, r := range chat.RemindersAbout(a.Store, chat.RecordAbout(ev.ID)) {
			if strings.HasPrefix(fmt.Sprint(r.Fields["title"]), title) {
				return ""
			}
		}
		return "no reminder " + title + "... about the meeting's page"
	}
	return []request{
		{name: "meeting-notes", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "event", map[string]any{"title": "Pricing review", "starts": at(time.Now().AddDate(0, 0, -1).Format("2006-01-02"), 10, 0)})
		}, say: "Write up yesterday's Pricing review meeting from my notes: we talked through the new plans. Decided to raise Pro to £12 a month. Decided to drop the free trial. I need to update the pricing page, and I need to email our customers about it.",
			check: func(a *app.App, _ string) string {
				ev := find(a, "event", "title", "pricing review")
				page, email := find(a, "task", "title", "pricing page"), find(a, "task", "title", "email")
				return all(want(len(list(a, "event")) == 1, "another event was made"),
					want(fmt.Sprint(ev.Fields["summary"]) != "" && ev.Fields["summary"] != nil, "the meeting has no summary"),
					want(has(ev, "decisions", "12") && has(ev, "decisions", "trial"), fmt.Sprintf("its decisions are %v", ev.Fields["decisions"])),
					want(page != nil && email != nil, "the two tasks were not made"),
					want(page != nil && page.Fields["event"] == ev.ID, "the task does not say which meeting it came up at"))
			}},
		{name: "meeting-recording", seed: func(t *testing.T, a *app.App) { recording(t, a, "Monday stand-up") },
			say: "Write up the meeting in my Monday stand-up recording.",
			check: func(a *app.App, _ string) string {
				file := find(a, "file", "title", "stand-up")
				var ev *store.Record
				for _, e := range list(a, "event") {
					if e.Fields["recording"] == file.ID {
						ev = e
					}
				}
				if ev == nil {
					return "no meeting keeps the recording"
				}
				task := find(a, "task", "title", "client")
				return all(want(has(ev, "decisions", "friday") && has(ev, "decisions", "-at-12"), fmt.Sprintf("its decisions are %v", ev.Fields["decisions"])),
					want(task != nil && task.Fields["event"] == ev.ID, "no task to email the client that came up at it"))
			}},
		{name: "meeting-link", seed: func(t *testing.T, a *app.App) {
			recording(t, a, "Call audio 14 Nov")
			mustCreate(t, a, "event", map[string]any{"title": "Weekly call", "starts": at(time.Now().AddDate(0, 0, -1).Format("2006-01-02"), 9, 0)})
		}, say: "The Call audio 14 Nov recording is of my Weekly call meeting yesterday. Write that meeting up from it.",
			check: func(a *app.App, _ string) string {
				ev, file := find(a, "event", "title", "weekly call"), find(a, "file", "title", "call audio")
				return all(want(len(list(a, "event")) == 1, "another event was made"),
					want(ev.Fields["recording"] == file.ID, "the meeting does not keep its recording"),
					want(has(ev, "decisions", "friday"), fmt.Sprintf("its decisions are %v", ev.Fields["decisions"])),
					want(find(a, "task", "title", "client") != nil, "no task to email the client"))
			}},
		{name: "meeting-record", seed: review, say: "I want the audio of my Design review meeting on Tuesday recorded. Set that up.",
			check: func(a *app.App, _ string) string { return asksToRecord(a, "Record ") }},
		{name: "meeting-app", seed: review, say: "My Design review on Tuesday is on Teams, and Teams records it. Set it up so its recording gets added here afterwards.",
			check: func(a *app.App, _ string) string { return asksToRecord(a, "Add the recording") }},
	}
}
