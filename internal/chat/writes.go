package chat

import (
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// The chat writes its own records (its chats, messages and questions) and
// what a tool changes through records.Apply, like every other way in, so
// each write keeps what was there and many go in together or not at all.
// What the person did here is logged as theirs: the owner's, or, for a
// service opened For someone else, that person's by name, never a bare
// "human" whoever it was.

// put writes one op and answers with the record as it now is, nil when
// the op removed it.
func (s *Service) put(op records.Op) (*store.Record, error) {
	done, err := s.Apply(op)
	if err != nil {
		return nil, err
	}
	last := done[len(done)-1]
	if last.After == nil {
		return nil, nil
	}
	return s.Store.Get(last.Type, last.ID)
}

// byPerson logs a change as the person this service speaks for.
func (s *Service) byPerson(c records.Change) string {
	c.By, c.Via, c.ByLogin = s.who.Who(), s.who.Device, s.who.Login
	return records.Record(s.Store, "human", c)
}
