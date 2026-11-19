package chat

import (
	"bytes"
	"fmt"
	"net/mail"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An email can carry an event as a real invitation (export.Invite): the
// mail app of whoever gets it shows its own Yes, No and Maybe and puts it
// in their calendar, with nothing of Sameway's to open. Its organiser is
// the address Sameway reads, so their answer comes back here, and is kept
// on the event beside who was invited, only from an address it was sent
// to: logged, so an action can wait for it, and told to the owner. Sent
// again it moves the event in their calendar; cancelled, takes it away.

// invitedState is how an invitation stands before an answer.
const invitedState = "invited"

// inviteOf is the event an email invites to, checked: one here, with a
// day.
func (s *Service) inviteOf(id string) (*store.Record, error) {
	t, ok := s.Store.Types().Get(records.EventType)
	if !ok {
		return nil, fmt.Errorf("this workspace has no events")
	}
	rec, err := s.Store.Get(records.EventType, id)
	if err != nil {
		return nil, fmt.Errorf("invite names no event here: give an event's id")
	}
	if !export.Dated(t, rec) {
		return nil, fmt.Errorf("%s has no day to invite to", t.Called(rec.ID, rec.Fields))
	}
	return rec, nil
}

// inviteWords is what an invitation does, for the owner's question.
func (s *Service) inviteWords(e emailArgs) string {
	rec, err := s.inviteOf(e.Invite)
	if err != nil {
		return ""
	}
	t, _ := s.Store.Types().Get(records.EventType)
	name := records.Name(s.Store, t, rec)
	if e.Cancel {
		return fmt.Sprintf("It takes %q out of their calendar.", name)
	}
	return fmt.Sprintf("It invites them to %q: their mail app offers Yes, No and Maybe and puts it in their calendar, and their answer comes back to the event here.", name)
}

// invitation is the calendar an email carries, and keeps on the event
// who it went to, unanswered, and that it was sent once more.
func (s *Service) invitation(e emailArgs) ([]byte, error) {
	rec, err := s.inviteOf(e.Invite)
	if err != nil {
		return nil, err
	}
	t, _ := s.Store.Types().Get(records.EventType)
	seq := 0
	switch n := rec.Fields["sequence"].(type) {
	case int64:
		seq = int(n) + 1
	case float64:
		seq = int(n) + 1
	}
	var b bytes.Buffer
	if err := export.Invite(&b, t, rec, s.MailFrom(), []string{e.To}, seq, e.Cancel); err != nil {
		return nil, err
	}
	state := invitedState
	if e.Cancel {
		state = "cancelled"
	}
	answers := withAnswer(records.StringList(rec.Fields["answers"]), e.To, state)
	_, err = s.Apply(records.Op{Type: records.EventType, ID: rec.ID, After: map[string]any{"sequence": seq, "answers": answers}})
	return b.Bytes(), err
}

// withAnswer is a list of answers with who's set to state, theirs kept in
// its place, or added.
func withAnswer(list []string, who, state string) []any {
	out := make([]any, 0, len(list)+1)
	found := false
	for _, l := range list {
		if sameAddress(answerWho(l), who) {
			l, found = answerWho(l)+": "+state, true
		}
		out = append(out, l)
	}
	if !found {
		out = append(out, who+": "+state)
	}
	return out
}

// answerWho is who an answer is from: "Pat Lee <pat@example.com>: accepted".
func answerWho(line string) string {
	if i := strings.LastIndex(line, ": "); i >= 0 {
		return line[:i]
	}
	return line
}

func sameAddress(a, b string) bool {
	x, err1 := mail.ParseAddress(a)
	y, err2 := mail.ParseAddress(b)
	return err1 == nil && err2 == nil && strings.EqualFold(x.Address, y.Address)
}

// InvitationAnswered keeps answers to invitations that came back by email
// from from on their events: only from an address an invitation went to,
// which is what makes it theirs to answer, and only their own answer, as
// one guest could otherwise decline for another.
func (s *Service) InvitationAnswered(answers []convert.Answer, from string) {
	t, ok := s.Store.Types().Get(records.EventType)
	if !ok {
		return
	}
	recs, _ := s.Store.List(records.EventType, store.ListOptions{})
	for _, a := range answers {
		if !sameAddress(a.Who, from) {
			continue
		}
		for _, rec := range recs {
			if uid, _ := rec.Fields["uid"].(string); uid != a.UID && rec.ID+"@sameway" != a.UID {
				continue
			}
			s.answer(t, rec, a)
		}
	}
}

func (s *Service) answer(t interface {
	Called(string, map[string]any) string
}, rec *store.Record, a convert.Answer) {
	list := records.StringList(rec.Fields["answers"])
	who := ""
	for _, l := range list {
		if sameAddress(answerWho(l), a.Who) {
			who = answerWho(l)
		}
	}
	if who == "" {
		return
	}
	done, err := s.Apply(records.Op{Type: records.EventType, ID: rec.ID, After: map[string]any{"answers": withAnswer(list, who, a.State)}})
	if err != nil {
		return
	}
	name := t.Called(rec.ID, rec.Fields)
	records.Record(s.Store, "system", records.Change{Action: "answered", Component: records.EventType, ID: rec.ID, Detail: who + " " + a.State, Ops: done})
	said := fmt.Sprintf("%s %s your invitation to %q.", who, a.State, name)
	s.Say(said)
	if s.Automate != nil && s.Automate.Tell != nil {
		s.Automate.Tell("They answered your invitation", said, "/t/"+records.EventType+"/"+rec.ID)
	}
}
