package chat

import (
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What was left over. An agent trying something added a list of habits
// that matched none, the assistant a calendar for a type no record of
// ever filled, and days later they were still on the person's tab,
// showing nothing, read by every model as part of the page. Neither can
// be shown wrong or repeats another, so nothing said them. A block that
// has shown nothing since a model put it there, days ago, is said now:
// it may be waiting to fill, or left over; the model asks the person.

// leftoverAfter is how long a block shows nothing before it is said.
const leftoverAfter = 7 * 24 * time.Hour

// leftoverProblems are Layout now's lines for blocks an agent or the
// assistant added at least a week ago that show nothing yet.
func (s *Service) leftoverProblems(recs []*store.Record, shows map[string]string) []string {
	now := s.Now()
	var out []string
	for _, b := range recs {
		by, _ := b.Fields["created_by"].(string)
		if by != "agent" && by != "assistant" || b.CreatedAt.IsZero() || now.Sub(b.CreatedAt) < leftoverAfter {
			continue
		}
		said, ok := shows[b.ID]
		if !ok || !strings.HasPrefix(said, "nothing yet") {
			continue
		}
		why := ""
		if _, rest, ok := strings.Cut(said, ": "); ok {
			why = " (" + rest + ")"
		}
		out = append(out, fmt.Sprintf("block %s (%s) has shown nothing since %s added it %d days ago%s: if it is not waiting for something the person will add, it may be left over; propose taking it away",
			b.ID, placedOf(b).called(), map[string]string{"agent": "an agent", "assistant": "you"}[by], int(now.Sub(b.CreatedAt).Hours()/24), why))
	}
	return out
}
