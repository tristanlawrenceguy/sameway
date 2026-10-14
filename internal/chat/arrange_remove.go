package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Taking away, in the same change as laying out. Told a tab had a block
// that could not be shown and one the same as another, a model laid the
// tab out with arrange_canvas, which refused it for leaving those two out,
// so it listed them again and they stayed; or it removed them first, and
// the person had two changes to undo for one tidy. remove takes them in
// the arrangement: one change, one Undo.

// arrangeProblems is why an arrangement cannot be made: a block not on the
// tab, listed twice, both listed and removed, or left out without being
// removed.
func arrangeProblems(before []*store.Record, byID map[string]*store.Record, items []arrangeItem, remove []string) []string {
	var problems []string
	listed := map[string]bool{}
	for _, it := range items {
		switch {
		case byID[it.ID] == nil:
			problems = append(problems, fmt.Sprintf("%s is not a block on this tab", it.ID))
		case listed[it.ID]:
			problems = append(problems, fmt.Sprintf("%s is listed twice", it.ID))
		}
		listed[it.ID] = true
	}
	gone := map[string]bool{}
	for _, id := range remove {
		switch {
		case byID[id] == nil:
			problems = append(problems, fmt.Sprintf("%s in remove is not a block on this tab", id))
		case listed[id]:
			problems = append(problems, fmt.Sprintf("%s is both listed and in remove; keep it in one", id))
		}
		gone[id] = true
	}
	for _, b := range before {
		if r := lookOf(b).region; !listed[b.ID] && !gone[b.ID] && r != "header" && r != "footer" {
			problems = append(problems, fmt.Sprintf("%s (%s %s) is missing; list every block, even one that stays where it is, or put it in remove to take it away", b.ID, placedOf(b).name(), placedOf(b).comp))
		}
	}
	return problems
}

// takenSaid is ", and 2 blocks taken away", or "".
func takenSaid(n int) string {
	switch n {
	case 0:
		return ""
	case 1:
		return ", and 1 block taken away"
	}
	return fmt.Sprintf(", and %d blocks taken away", n)
}
