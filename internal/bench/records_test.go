package bench

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
)

// recordWork is the plain work on records: several made at once, a thing
// found by a word in it and read, and a deletion taken back.
func recordWork() []request {
	return []request{
		{name: "brain-dump", say: "Add these tasks: call the plumber, renew my passport, buy stamps, book the dentist",
			check: func(a *app.App, _ string) string {
				var missing []string
				for _, w := range []string{"plumber", "passport", "stamps", "dentist"} {
					if find(a, "task", "title", w) == nil {
						missing = append(missing, w)
					}
				}
				return all(want(len(missing) == 0, "no task for: "+strings.Join(missing, ", ")),
					want(len(list(a, "task")) == 4, fmt.Sprintf("%d tasks, not 4", len(list(a, "task")))))
			}},
		{name: "wifi-word", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "note", map[string]any{"title": "Flat", "body": "Bins go out on Tuesday.\n\nThe landlord is Mr Okafor.\n\nRouter is under the stairs; the wifi password is heron-4172."})
			mustCreate(t, a, "note", map[string]any{"title": "Book club", "body": "Next: Middlemarch."})
			mustCreate(t, a, "task", map[string]any{"title": "Fix the router shelf"})
		}, say: "What's my wifi password?",
			check: func(_ *app.App, reply string) string {
				return want(strings.Contains(reply, "heron-4172"), "it did not find the password: "+reply)
			}},
		{name: "undo-delete", seed: func(t *testing.T, a *app.App) {
			mustCreate(t, a, "note", map[string]any{"title": "Recipe for soup", "body": "Leeks, potatoes, stock."})
			mustCreate(t, a, "note", map[string]any{"title": "Holiday ideas", "body": "Lisbon in May."})
			args, _ := json.Marshal(map[string]any{"type": "note", "id": find(a, "note", "title", "soup").ID})
			if text, bad := a.Chat.Call("delete_record", args); bad {
				t.Fatal(text)
			}
		}, say: "I didn't mean to delete the soup recipe. Bring it back.",
			check: func(a *app.App, reply string) string {
				return all(want(has(find(a, "note", "title", "soup"), "body", "leeks"), "the soup recipe is not back: "+reply),
					want(find(a, "note", "title", "holiday") != nil, "the other note went"))
			}},
	}
}
