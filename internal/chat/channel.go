package chat

import (
	"context"
	"fmt"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A conversation on a messaging channel, such as the owner's Telegram
// bot, with someone the owner has not let in: answered as a visitor to
// the published pages is, with their reach and nothing more, what they
// ask to be made a request on Today, the owner's answer told back where
// they asked (Service.Channel). What was said before comes with it, so a
// "Tuesday then" follows on from "when could you come?".

// Said is one message of a channel's conversation.
type Said struct {
	Ours bool   `json:"ours"`
	Text string `json:"text"`
}

// AnswerChat answers the latest message of a conversation on a channel,
// where, as whoever this service speaks for.
func (s *Service) AnswerChat(ctx context.Context, where string, said []Said) (string, error) {
	if s.Provider == nil {
		return "", fmt.Errorf("no assistant to answer just now")
	}
	owner := s.Owner.Name
	if owner == "" {
		owner = "the owner"
	}
	system := fmt.Sprintf("You answer %s on %s, for %s, as %s's assistant. You hold their reach, nothing more: you read what %s published. Answer in a few plain sentences from what you read; search before saying you cannot answer, and when you cannot, say so. When they ask for something to be made, such as a time to meet (an event, with when it starts and their name in the title), make it with create_record: it is not made but put to %s, who says yes or no, and they hear back here; say so, and never say it is done. Never say you did anything you did not do with a tool. %s",
		s.who.Who(), where, owner, owner, owner, owner, records.DataNotInstructions)
	var msgs []llm.Message
	for _, m := range said {
		role := llm.RoleUser
		if m.Ours {
			role = llm.RoleAssistant
		}
		if n := len(msgs); n > 0 && msgs[n-1].Role == role {
			msgs[n-1].Content += "\n\n" + m.Text
			continue
		}
		msgs = append(msgs, llm.Message{Role: role, Content: m.Text})
	}
	if len(msgs) == 0 || msgs[0].Role != llm.RoleUser {
		return "", fmt.Errorf("nothing to answer")
	}
	defer s.tellOwner(s.Now())
	return s.answerWith(ctx, system, msgs)
}
