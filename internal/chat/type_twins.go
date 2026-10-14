package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Records that are one thing twice. A person who let a week pile up found
// "Dentist appointment" twice among their notes and "Pick up dry cleaning"
// twice, and the assistant, asked to sort the week out, sorted around
// them: it is told of a twin only as it makes one (same_title.go), and
// these were made before. A list page offers to keep one of each; the
// type index now says them to the model too, with keep_one to make them
// one when the person agrees.

// twinsMost is how many repeated names the index says by name.
const twinsMost = 3

// twinsSaid is " 2 names are on more than one open note with nothing to
// tell them apart: Dentist appointment (2), Pick up dry cleaning (2); ...",
// or "" when there are none.
func (s *Service) twinsSaid(t *schema.Type) string {
	groups := records.Twins(s.Store, t)
	if len(groups) == 0 {
		return ""
	}
	var said []string
	for _, g := range groups[:min(twinsMost, len(groups))] {
		said = append(said, fmt.Sprintf("%s (%d)", t.Called(g[0].ID, g[0].Fields), len(g)))
	}
	if len(groups) > twinsMost {
		said = append(said, fmt.Sprintf("and %d more", len(groups)-twinsMost))
	}
	names := "names are"
	if len(groups) == 1 {
		names = "name is"
	}
	return fmt.Sprintf(" %d %s on more than one open %s with nothing to tell them apart: %s; when the person agrees they are one thing, keep_one makes them one.",
		len(groups), names, schema.Words(t.Name), strings.Join(said, ", "))
}
