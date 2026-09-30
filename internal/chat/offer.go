package chat

import (
	"errors"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Outward says whether changing a setting acts beyond the workspace and so
// cannot be taken back once it has (publishing, letting in, updating); the
// rest, such as which parts a page shows, are undone like anything else.
func Outward(key string) bool { return outward[key] != nil }

// Offer puts a question to the owner that Sameway asks of itself, having
// watched how the workspace is used: what it would change, the reason in
// one sentence, and its own Yes and No. It is a question like the ones the
// assistant asks, answered the same way, and what a Yes changes is logged
// and can be undone.
func (s *Service) Offer(ask, why, yes, no string, action map[string]any) (*store.Record, error) {
	r := s.ask(question{ask: ask, detail: why, yes: yes, no: no}, action)
	if r.isErr || r.change == nil {
		return nil, errors.New(r.text)
	}
	return s.Store.Get(ProposalType, r.change.ID)
}
