package store

import (
	"fmt"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// OnChange is told each record made, changed or deleted here, with what it
// was (nil when it is new) and what it is (nil when it is gone), so
// something can happen when a record comes to be a certain way: a task
// done, a file arrived, a device switched on (chat/automate.go). Records
// that arrive from another computer are left to that computer.

func (s *Store) changed(t *schema.Type, was, now *Record) {
	if s.OnChange != nil {
		s.OnChange(t, was, now)
	}
}

// before is a record as it is, for OnChange, when something listens.
func (s *Store) before(t *schema.Type, id string) *Record {
	if s.OnChange == nil {
		return nil
	}
	rec, err := s.Get(t.Name, id)
	if err != nil {
		return nil
	}
	return rec
}

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
