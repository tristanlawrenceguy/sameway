package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Taking away, in the same change as laying out. Told a tab had a block
// that could not be shown and one the same as another, a model laid the
// tab out with arrange_canvas, which refused it for leaving those two out,
// so it listed them again and they stayed; or it removed them first, and
// the person had two changes to undo for one tidy. remove takes them in
// the arrangement: one change, one Undo.

// arrangeProblems is why an arrangement cannot be made: a block not on the
// tab, listed twice, or both listed and removed. One left out is not a
// problem: it keeps its order after the ones listed, and the answer says
// so (unlisted).
func arrangeProblems(byID map[string]*store.Record, items []arrangeItem, remove []string) []string {
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
	return problems
}

// unlisted are the blocks an arrangement left out, neither listed nor
// removed, in their order; the header's and footer's are their own.
func unlisted(before []*store.Record, items []arrangeItem, remove []string) []*store.Record {
	named := map[string]bool{}
	for _, it := range items {
		named[it.ID] = true
	}
	for _, id := range remove {
		named[id] = true
	}
	var out []*store.Record
	for _, b := range before {
		if r := lookOf(b).region; !named[b.ID] && r != "header" && r != "footer" {
			out = append(out, b)
		}
	}
	return out
}

// unlistedSaid is what the answer says of the blocks left out: where they
// went, and how to place them.
func unlistedSaid(left []*store.Record) string {
	if len(left) == 0 {
		return ""
	}
	var names []string
	for _, b := range left {
		names = append(names, b.ID+" ("+placedOf(b).name()+" "+placedOf(b).comp+")")
	}
	return fmt.Sprintf(". %d not listed kept their order after the ones listed: %s; list them to place them, or put one in remove to take it away", len(left), schema.Names(names))
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
