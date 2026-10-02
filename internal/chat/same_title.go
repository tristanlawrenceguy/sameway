package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// sameTitle says when a record just made has the title of one there
// already: asked to organise the chapters a person has, a model made new
// ones of the same names, and the person's were left as they were.
func (s *Service) sameTitle(t *schema.Type, rec *store.Record) string {
	title := strings.ToLower(strings.TrimSpace(recordTitle(s.Store, t, rec)))
	if title == "" {
		return ""
	}
	recs, _ := s.Store.List(t.Name, store.ListOptions{})
	for _, r := range recs {
		if r.ID != rec.ID && strings.ToLower(strings.TrimSpace(recordTitle(s.Store, t, r))) == title {
			return fmt.Sprintf(" A %s titled the same was there already, %s: if the person meant that one, take this back with undo_change and use it.", t.Name, r.ID)
		}
	}
	return ""
}
