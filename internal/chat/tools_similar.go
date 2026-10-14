package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A block added beside one that already shows the same records. Asked for
// a habit tracker on a tab that had one, a model added a second, and the
// tab showed every habit twice; the rule to update rather than add a near
// duplicate was in its prompt, and what was already on the tab was a list
// of props it did not read for that. Said in the add's own answer, with
// what to do, it is in front of the model the moment it matters. The add
// is not refused: two lists of tasks, one this week's and one overdue, are
// often what the person asked for.

// recordSource is the components that show records, by the prop that
// names which: a block of one of these and the same records is similar.
var recordSource = map[string]string{"collection": "type", "calendar": "type", "chart": "type", "tracker": "habits", "record": "id"}

// similarOn is what the add says when the tab already has a block of the
// same component over the same records: which, and how to use it instead.
func (s *Service) similarOn(canvas, name string, props map[string]any, newID string) string {
	key, ok := recordSource[name]
	if !ok {
		return ""
	}
	blocks, err := s.Store.List(records.BlockType, store.ListOptions{OrderBy: "position"})
	if err != nil {
		return ""
	}
	var same []string
	for _, b := range records.OnCanvas(blocks, canvas) {
		if b.ID == newID || b.Fields["component"] != name {
			continue
		}
		had, _ := b.Fields["props"].(map[string]any)
		if name != "tracker" && fmt.Sprint(had[key]) != fmt.Sprint(props[key]) {
			continue
		}
		what := records.Summarise(name, had)
		if what == "" {
			what, _ = had["label"].(string)
		}
		if what == "" {
			what = name
		}
		same = append(same, fmt.Sprintf("%s (%s)", b.ID, what))
	}
	if len(same) == 0 {
		return ""
	}
	return fmt.Sprintf("\nThis tab already had a %s of the same records: %s. If the person meant that one changed, take this back with undo_change and use update_component on it; keep both only when they show different things.", name, strings.Join(same, ", "))
}
