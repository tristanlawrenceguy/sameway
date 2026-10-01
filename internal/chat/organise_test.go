package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// A piece is organised into parts in reading order, with its material
// beside it, in one change that one Undo takes back.
func TestWritingIsOrganisedIntoParts(t *testing.T) {
	svc := newFullService(t)
	note := func(title string) string {
		r, err := svc.Store.Create("note", map[string]any{"title": title})
		if err != nil {
			t.Fatal(err)
		}
		return r.ID
	}
	book, one, two, three, guide := note("The Pond"), note("Digging"), note("Lining"), note("Filling"), note("Submission guidelines")
	run(t, svc, "organise_writing", map[string]any{"piece": book, "parts": []any{two, one}, "material": []any{map[string]any{"id": guide}}})
	run(t, svc, "organise_writing", map[string]any{"piece": book, "parts": []any{three}})

	nt, _ := svc.Store.Types().Get("note")
	piece, _ := svc.Store.Get("note", book)
	var titles []string
	for _, p := range chat.Parts(svc.Store, nt, piece) {
		titles = append(titles, p.Fields["title"].(string))
	}
	if len(titles) != 3 || titles[0] != "Filling" || titles[1] != "Lining" || titles[2] != "Digging" {
		t.Errorf("parts named come first, in order, then the rest as they were: %v", titles)
	}
	part, _ := svc.Store.Get("note", one)
	if p := chat.PieceOf(svc.Store, nt, part); p == nil || p.ID != book {
		t.Error("a part knows its piece")
	}
	mine, above := chat.Material(svc.Store, nt, part)
	if len(mine) != 0 || len(above[book]) != 1 || above[book][0].ID != guide {
		t.Errorf("a part has the whole piece's material: %v %v", mine, above)
	}
	refused(t, svc, "organise_writing", map[string]any{"piece": one, "parts": []any{book}}, "inside it")

	last := entries(t, svc, "organised")
	run(t, svc, "undo_change", map[string]any{"id": last[len(last)-1].ID})
	if p, _ := svc.Store.Get("note", three); p.Fields["part_of"] != "" && p.Fields["part_of"] != nil {
		t.Errorf("undo takes the second organising back: %v", p.Fields["part_of"])
	}
	if p, _ := svc.Store.Get("note", two); p.Fields["part_of"] != book {
		t.Error("and leaves the first")
	}
}
