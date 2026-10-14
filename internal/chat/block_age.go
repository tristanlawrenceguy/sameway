package chat

import (
	"fmt"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// When a block was added, and by whom. The canvas the model reads gave
// each block as its props, so a "click me" button an agent left a week ago
// to test something read the same as the list the person asked for this
// morning; a model judging what a tab is for, and what is left over, had
// nothing to judge by. Each block now says how long it has been there and
// who put it there.

// addedSaid is " added 9 days ago by an agent", or "" when it is not known.
func addedSaid(blk *store.Record, now time.Time) string {
	if blk.CreatedAt.IsZero() {
		return ""
	}
	when := "today"
	switch days := int(now.Sub(blk.CreatedAt).Hours() / 24); {
	case days == 1:
		when = "yesterday"
	case days > 1:
		when = fmt.Sprintf("%d days ago", days)
	}
	by := map[string]string{"assistant": "the assistant", "agent": "an agent", "system": "Sameway"}[fmt.Sprint(blk.Fields["created_by"])]
	if by == "" {
		return " added " + when
	}
	return " added " + when + " by " + by
}
