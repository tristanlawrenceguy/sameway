package store

import (
	"fmt"
	"time"
)

// DeleteAll removes every record of a type.
func (s *Store) DeleteAll(typeName string) error {
	t, err := s.typ(typeName)
	if err != nil {
		return err
	}
	recs, _ := s.List(t.Name, ListOptions{})
	for _, r := range recs {
		s.stamp(t, r.ID, nil, nil, time.Time{})
	}
	_, err = s.db.Exec(fmt.Sprintf("DELETE FROM %s", quote(t.Name)))
	return err
}
