package chat

import (
	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A habit's standing in words, for an agent as for the person: logging
// says where it stands now, as the Log press does, and reading a habit
// says how it has gone, so an agent never works the numbers out from
// every entry by hand.

// habitSaid is what an entry just made does to its habit: " Water now
// stands at: 6 of 8 glasses today. …", or "".
func (s *Service) habitSaid(t *schema.Type, rec *store.Record) string {
	if t.Name != records.EntryType {
		return ""
	}
	id, _ := rec.Fields["habit"].(string)
	habit, err := s.Store.Get(records.HabitType, id)
	if err != nil {
		return ""
	}
	return " " + blocks.HabitOf(habit).Name + " now: " + blocks.HabitSaid(s.Store, habit, s.Now())
}

// standing is a habit's standing for get_record, or "" for anything else.
func (s *Service) standing(t *schema.Type, rec *store.Record) string {
	if t.Name != records.HabitType {
		return ""
	}
	return blocks.HabitSaid(s.Store, rec, s.Now())
}
