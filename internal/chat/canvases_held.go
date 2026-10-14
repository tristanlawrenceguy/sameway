package chat

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// What each tab holds, in the tab list the model reads. The tabs were
// given as names and addresses, so a model asked for a task list added a
// third one on Home while Work already had two, kept making tabs while an
// empty one sat beside them, and could not say where the person's habits
// were. Each tab now says what is on it, by the names its blocks go by.

// heldMost is how many blocks a tab names before it says how many more.
const heldMost = 6

// tabHolds is ": Tasks due this week, Habits, and 3 more", or ": empty".
func (s *Service) tabHolds(canvas string) string {
	var names []string
	n := 0
	for _, b := range s.canvasBlocks(canvas) {
		if b.Fields["created_by"] == "system" {
			continue // the tab's own chat and search, on every tab
		}
		n++
		if len(names) == heldMost {
			continue
		}
		names = append(names, placedOf(b).name()) // one name for a block, as Layout now gives it
	}
	if n == 0 {
		return ": empty"
	}
	out := ": " + schema.Names(names)
	if n > len(names) {
		out += fmt.Sprintf(", and %d more", n-len(names))
	}
	return out
}
