package store

import (
	"fmt"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// Listen adds fn to what is told of each record made, changed or deleted
// here, with what it was (nil when it is new) and what it is (nil when it
// is gone), so something can happen when a record comes to be a certain
// way: a task done, a file arrived, a device switched on
// (chat/automate.go). Every write passes here, whichever way it came;
// records that arrive from another computer are left to that computer.
// A change as a whole, once per change, is told by records.Listen.
func (s *Store) Listen(fn func(t *schema.Type, was, now *Record)) {
	s.listening.Lock()
	defer s.listening.Unlock()
	s.listeners = append(s.listeners, fn)
}

func (s *Store) changed(t *schema.Type, was, now *Record) {
	for _, fn := range s.heard() {
		fn(t, was, now)
	}
}

func (s *Store) heard() []func(t *schema.Type, was, now *Record) {
	s.listening.Lock()
	defer s.listening.Unlock()
	return s.listeners
}

// before is a record as it is, for the listeners, when something listens.
func (s *Store) before(t *schema.Type, id string) *Record {
	if len(s.heard()) == 0 {
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
