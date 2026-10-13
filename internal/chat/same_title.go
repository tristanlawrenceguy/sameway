package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// sameTitle says when a record just made is called as open ones already
// there are (records.Alike), with the day of each, so the model can judge:
// asked to organise the chapters a person has, a model made new ones of
// the same names; asked again for groceries, it made a sixth task.
func (s *Service) sameTitle(t *schema.Type, rec *store.Record) string {
	alike := records.Alike(s.Store, t, rec)
	if len(alike) == 0 {
		return ""
	}
	return fmt.Sprintf(" %s called the same %s there already, still open: %s. If the person meant one of those, take this back with undo_change and use it; if they are one thing twice, keep_one makes them one.",
		schema.Count(len(alike), t.Name), map[bool]string{true: "was", false: "were"}[len(alike) == 1], s.alikeSaid(t, alike))
}

// alikeSaid is records alike by id, up to three, each with its day.
func (s *Service) alikeSaid(t *schema.Type, alike []*store.Record) string {
	day := t.DayField()
	var said []string
	for _, r := range alike[:min(3, len(alike))] {
		w := r.ID
		if d, _ := r.Fields[day].(string); day != "" && d != "" {
			w += " (" + t.FieldWords(day) + " " + when.Text(d, s.Now().Location(), false) + ")"
		}
		said = append(said, w)
	}
	more := ""
	if len(alike) > 3 {
		more = fmt.Sprintf(" and %d more", len(alike)-3)
	}
	return strings.Join(said, ", ") + more
}
