package chat

import (
	"errors"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A chat cleared or deleted is kept on its entry with its messages, and
// put back from there.

// keptMessages is a chat's messages as the log keeps them, to put back.
func keptMessages(msgs []*store.Record) []any {
	out := make([]any, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, map[string]any{"id": m.ID, "fields": m.Fields})
	}
	return out
}

// inverseChat puts back a chat that was cleared or deleted: the chat
// itself when it was deleted, then each message that is not there.
func (s *Service) inverseChat(id string, before map[string]any) (func() (Change, error), error) {
	msgs := blocksIn(map[string]any{"blocks": before["messages"]})
	conv, _ := before["conversation"].(map[string]any)
	if len(msgs) == 0 && conv == nil {
		return nil, errors.New("the entry does not say what was in the chat")
	}
	if _, err := s.Store.Get(ConversationType, id); err == nil && conv != nil {
		return nil, errors.New("it is already back")
	}
	return func() (Change, error) {
		if conv != nil {
			if _, err := s.Store.Restore(ConversationType, id, conv); err != nil {
				return Change{}, err
			}
		}
		for _, m := range msgs {
			if _, err := s.Store.Get(MessageType, m.id); err == nil {
				continue
			}
			if _, err := s.Store.Restore(MessageType, m.id, m.fields); err != nil {
				return Change{}, err
			}
		}
		title, _ := conv["title"].(string)
		return Change{Action: "restored", Component: "conversation", ID: id, Detail: title}, nil
	}, nil
}
