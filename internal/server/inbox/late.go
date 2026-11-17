package inbox

import (
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A conversation waiting on them is late once their answer was due: on
// the day they said or the owner set (the email's expected), or three days
// after it was sent when nobody said. Late is a change like any, logged,
// so an action can wait for it (when changed, what email, turn=late),
// such as the assistant writing a nudge in the thread, and the owner is
// told. Their next email decides the turn again.

// lateAfter is how long an answer may take when nobody said.
const lateAfter = 3 * 24 * time.Hour

// MarkLate marks each conversation waiting on them past when their answer
// was due. It is a background job's work.
func (s *Service) MarkLate(now time.Time) error {
	if _, ok := s.app.Types.Get(records.EmailType); !ok {
		return nil
	}
	emails, err := s.app.Store.List(records.EmailType, store.ListOptions{})
	if err != nil {
		return err
	}
	for _, e := range emails {
		if e.Fields["turn"] != "theirs" || !now.After(dueBy(e)) {
			continue
		}
		done, err := records.ApplyOps(s.app.Store, records.Op{Type: records.EmailType, ID: e.ID, After: map[string]any{"turn": "late"}})
		if err != nil {
			continue
		}
		who := s.nameOf(e)
		records.Record(s.app.Store, "system", records.Change{Action: "marked late", Component: records.EmailType, ID: e.ID, Detail: who, Ops: done})
		if s.app.Automate != nil && s.app.Automate.Tell != nil {
			s.app.Automate.Tell("An answer is late", "No answer yet to "+who+", since "+waitedFor(now.Sub(sentTime(e)))+".", "/t/email/"+e.ID+"?show="+ConversationPart+"#conversation")
		}
	}
	return nil
}

// dueBy is when an answer to an email was due.
func dueBy(e *store.Record) time.Time {
	if v, _ := e.Fields["expected"].(string); v != "" {
		if at, err := time.Parse(time.RFC3339, v); err == nil {
			return at
		}
	}
	return sentTime(e).Add(lateAfter)
}
