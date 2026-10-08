package records

import (
	"errors"
)

// A chat cleared or deleted is kept on its entry with its messages, and
// put back from there.

// inverseChat puts back a chat that was cleared or deleted: the chat
// itself when it was deleted, then each message that is not there.
func (b *Book) inverseChat(id string, before map[string]any) (func() (Change, error), error) {
	msgs := blocksIn(map[string]any{"blocks": before["messages"]})
	conv, _ := before["conversation"].(map[string]any)
	if len(msgs) == 0 && conv == nil {
		return nil, errors.New("the entry does not say what was in the chat")
	}
	if _, err := b.Store.Get(ConversationType, id); err == nil && conv != nil {
		return nil, errors.New("it is already back")
	}
	return func() (Change, error) {
		if conv != nil {
			if _, err := b.Store.Restore(ConversationType, id, conv); err != nil {
				return Change{}, err
			}
		}
		for _, m := range msgs {
			if _, err := b.Store.Get(MessageType, m.id); err == nil {
				continue
			}
			if _, err := b.Store.Restore(MessageType, m.id, m.fields); err != nil {
				return Change{}, err
			}
		}
		title, _ := conv["title"].(string)
		return Change{Action: "restored", Component: "conversation", ID: id, Detail: title}, nil
	}, nil
}
