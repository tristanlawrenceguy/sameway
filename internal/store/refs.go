package store

import (
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// checkRefs refuses a ref to a record that is not there, naming the field
// and the type, so a task cannot belong to a project that does not exist.
// An empty ref is none, which is allowed.
func (s *Store) checkRefs(t *schema.Type, clean map[string]any) error {
	for _, f := range t.Fields {
		if f.Type != "ref" {
			continue
		}
		id, _ := clean[f.Name].(string)
		if id == "" {
			continue
		}
		if _, err := s.Get(f.To, id); err != nil {
			return &schema.ValidationError{Problems: map[string]string{f.Name: fmt.Sprintf("no %s with id %s", f.To, id)}}
		}
	}
	return nil
}
