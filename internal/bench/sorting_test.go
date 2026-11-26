package bench

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// sorting is help with a person's own words: a judgement call on them is
// suggested and waits for the person, with the words as they were; a
// plain command (fix the spelling) is just done.
func sorting() []request {
	const letter = "I am writing to you in order to apply for the position of gardener. I have got a very large amount of experience in the field of gardening, due to the fact that I have worked in gardens for ten years."
	waiting := func(a *app.App, note string) int {
		n := 0
		for _, s := range list(a, records.SuggestionType) {
			if s.Fields["about"] == "note/"+note && s.Fields["state"] == "pending" {
				n++
			}
		}
		return n
	}
	return []request{
		{name: "suggest-wording", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "note", map[string]any{"title": "Cover letter", "body": letter})
		}, say: "Tighten the wording of my Cover letter note. I want to look at each change before it goes in.",
			check: func(a *app.App, _ string) string {
				n := find(a, "note", "title", "cover letter")
				if n == nil {
					return "no Cover letter"
				}
				return all(want(n.Fields["body"] == letter, "its words were changed before the person saw the changes"),
					want(waiting(a, n.ID) > 0, "no suggestion waits on it"))
			}},
		{name: "fix-spelling", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "note", map[string]any{"title": "Packing list", "body": "Bring the tent, the sleping bags and a torch. Remeber the mallet."})
		}, say: "Fix the spelling in my Packing list note",
			check: func(a *app.App, _ string) string {
				n := find(a, "note", "title", "packing list")
				if n == nil {
					return "no Packing list"
				}
				body := strings.ToLower(fmt.Sprint(n.Fields["body"]))
				return want(strings.Contains(body, "sleeping") && strings.Contains(body, "remember") && !strings.Contains(body, "sleping"),
					"the spelling was not fixed in the note: "+body)
			}},
	}
}
