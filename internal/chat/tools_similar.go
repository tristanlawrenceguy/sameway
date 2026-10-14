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
// same component over the same records: which, and how to use it instead;
// and, when another tab has one, where, since a person asking for their
// tasks "here" may have meant the list they already keep on Work.
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
	elsewhere := map[string][]string{}
	for _, b := range blocks {
		if b.ID == newID || b.Fields["component"] != name {
			continue
		}
		had, _ := b.Fields["props"].(map[string]any)
		if name != "tracker" && fmt.Sprint(had[key]) != fmt.Sprint(props[key]) {
			continue
		}
		what := fmt.Sprintf("%s (%s)", b.ID, similarCalled(name, had))
		if on, _ := b.Fields["canvas"].(string); on != canvas {
			elsewhere[on] = append(elsewhere[on], what)
			continue
		}
		same = append(same, what)
	}
	out := ""
	if len(same) > 0 {
		out = fmt.Sprintf("\nThis tab already had a %s of the same records: %s. If the person meant that one changed, take this back with undo_change and use update_component on it; keep both only when they show different things.", name, strings.Join(same, ", "))
	}
	var tabs []string
	for _, c := range s.Canvases() {
		if ids, ok := elsewhere[c.ID]; ok {
			tabs = append(tabs, fmt.Sprintf("%s has %s", c.Name, strings.Join(ids, ", ")))
		}
	}
	if len(tabs) > 0 {
		out += fmt.Sprintf("\nAnother tab shows the same records as a %s: %s. Keep this one when the person wants them here too; if they meant the one they already have, take this back with undo_change and link them to that tab.", name, strings.Join(tabs, "; "))
	}
	return out
}

// similarCalled is what a block is called, for saying it among others.
func similarCalled(name string, props map[string]any) string {
	if what := records.Summarise(name, props); what != "" {
		return what
	}
	if label, _ := props["label"].(string); label != "" {
		return label
	}
	return name
}
