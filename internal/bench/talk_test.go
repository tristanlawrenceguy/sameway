package bench

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// talks are the conversation's own parts as the assistant uses them: the
// chat moved to a pane and back, a question left waiting with its words,
// and a change to someone's writing suggested, not made.
func talks() []request {
	const body = "At the meeting it was decided by everyone that the garden opens in May. We will be needing to recieve the seeds before then."
	return []request{
		{name: "chat-pane", seed: func(t *testing.T, a *app.App) {
			chatIn(t, a, "main")
			blockOf(t, a, "collection", map[string]any{"type": "task", "label": "To do"})
		}, say: "Put the chat in a pane on the right, beside my tasks.",
			check: func(a *app.App, _ string) string {
				return want(chatRegion(a) == "right", "the chat is in "+chatRegion(a)+", not the right pane")
			}},
		{name: "chat-middle", seed: func(t *testing.T, a *app.App) {
			chatIn(t, a, "right")
			blockOf(t, a, "collection", map[string]any{"type": "task", "label": "To do"})
		}, say: "Move the chat out of the side pane, back into the middle of the page.",
			check: func(a *app.App, _ string) string {
				return want(chatRegion(a) == "main", "the chat is in "+chatRegion(a)+", not the middle")
			}},
		{name: "ask-waits", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "task", map[string]any{"title": "Pay rent", "notes": "Landlord wants it by the 1st."})
			mustCreate(t, a, "task", map[string]any{"title": "Book the dentist"})
		}, say: "Delete the notes field from my tasks. I never use it.",
			check: func(a *app.App, _ string) string {
				if !has(find(a, "task", "title", "pay rent"), "notes", "landlord") {
					return "the notes went before the question was answered"
				}
				for _, p := range list(a, records.ProposalType) {
					if p.Fields["state"] == "pending" && has(p, "summary", "delete notes from every task") {
						return all(want(has(p, "detail", "1 of 2 have something in it"), "the question does not say what would be lost: "+fmt.Sprint(p.Fields["detail"])),
							want(has(p, "yes", "delete it") && has(p, "no", "keep it") && has(p, "instead", "hide it instead"), "the question lacks one of its three answers"))
					}
				}
				return "no question to delete notes waits"
			}},
		{name: "suggest-shown", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "note", map[string]any{"title": "Garden plan", "body": body})
		}, say: "Suggest how to tighten the wording of my note Garden plan. Do not change it yourself: I want to accept or decline each change.",
			check: func(a *app.App, _ string) string {
				n := find(a, "note", "title", "garden plan")
				if n == nil || n.Fields["body"] != body {
					return "the note was changed, not suggested to"
				}
				waiting := 0
				for _, s := range list(a, records.SuggestionType) {
					passage, _ := s.Fields["passage"].(string)
					if s.Fields["about"] == "note/"+n.ID && passage != "" && strings.Contains(body, passage) && fmt.Sprint(s.Fields["why"]) != "" &&
						s.Fields["state"] != "accepted" && s.Fields["state"] != "declined" {
						waiting++
					}
				}
				return want(waiting > 0, "no suggestion waits on the note with its passage and why")
			}},
	}
}

func chatIn(t *testing.T, a *app.App, region string) {
	f := a.Chat.BlockFields(map[string]any{"component": records.ComponentName, "props": map[string]any{}, "span": 12, "region": region})
	if _, err := a.Store.Create(records.BlockType, f); err != nil {
		t.Fatal(err)
	}
}

func chatRegion(a *app.App) string {
	for _, b := range list(a, records.BlockType) {
		if b.Fields["component"] == records.ComponentName {
			if r, _ := b.Fields["region"].(string); r != "" {
				return r
			}
			return "main"
		}
	}
	return "no chat"
}
