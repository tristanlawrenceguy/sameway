package chat

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// ConversationType is the content type that holds one chat: a run of
// messages with its own history. A person can have several and move
// between them; the one opened most recently is the one the chat shows,
// and the only one the model is told about.
const ConversationType = "conversation"

// Current is the id of the chat that is open: the one opened most
// recently, made when there is none yet.
func (s *Service) Current() string {
	if s.convo != "" {
		if _, err := s.Store.Get(ConversationType, s.convo); err == nil {
			return s.convo
		}
	}
	all := s.Conversations()
	if len(all) == 0 {
		rec, err := s.Store.Create(ConversationType, map[string]any{"opened": now()})
		if err != nil {
			return ""
		}
		s.convo = rec.ID
		return s.convo
	}
	s.convo = all[0].ID
	return s.convo
}

// Conversations lists every chat, the most recently opened first.
func (s *Service) Conversations() []*store.Record {
	recs, err := s.Store.List(ConversationType, store.ListOptions{})
	if err != nil {
		return nil
	}
	sort.SliceStable(recs, func(i, j int) bool {
		a, _ := recs[i].Fields["opened"].(string)
		b, _ := recs[j].Fields["opened"].(string)
		if a != b {
			return a > b
		}
		return recs[i].CreatedAt.After(recs[j].CreatedAt)
	})
	return recs
}

// NewChat starts a chat with nothing in it and opens it.
func (s *Service) NewChat() (*store.Record, error) {
	rec, err := s.Store.Create(ConversationType, map[string]any{"opened": now()})
	if err != nil {
		return nil, err
	}
	s.convo = rec.ID
	return rec, nil
}

// OpenChat makes the chat named the current one.
func (s *Service) OpenChat(id string) error {
	if _, err := s.Store.Update(ConversationType, id, map[string]any{"opened": now()}); err != nil {
		return err
	}
	s.convo = id
	return nil
}

// DeleteChat removes a chat and every message in it. When it was the
// current one, the most recently opened of the rest takes its place.
func (s *Service) DeleteChat(id string) error {
	msgs, err := s.MessagesIn(id)
	if err != nil {
		return err
	}
	for _, m := range msgs {
		s.Store.Delete(MessageType, m.ID)
	}
	if err := s.Store.Delete(ConversationType, id); err != nil {
		return err
	}
	if s.convo == id {
		s.convo = ""
	}
	return nil
}

// Messages are the messages of the current chat, oldest first.
func (s *Service) Messages() ([]*store.Record, error) {
	return s.MessagesIn(s.Current())
}

// Title names a chat: what it was named after, or the first thing the
// person said in it, for a chat from before chats had names.
func (s *Service) Title(c *store.Record) string {
	if title, _ := c.Fields["title"].(string); title != "" {
		return title
	}
	msgs, _ := s.MessagesIn(c.ID)
	for _, m := range msgs {
		if content, _ := m.Fields["content"].(string); m.Fields["role"] == "user" && strings.TrimSpace(content) != "" {
			return truncate(strings.TrimSpace(content), 60)
		}
	}
	return "New chat"
}

// MessagesIn are the messages of one chat, oldest first. A message from
// before there were several chats names none, and belongs to the oldest.
func (s *Service) MessagesIn(id string) ([]*store.Record, error) {
	recs, err := s.Store.List(MessageType, store.ListOptions{OrderBy: "created_at"})
	if err != nil {
		return nil, err
	}
	// The oldest chat: made first, or, made in the same second, opened
	// first, since the opened stamp is finer than the record's clock.
	oldest, firstOpened := "", ""
	var first time.Time
	for _, c := range s.Conversations() {
		opened, _ := c.Fields["opened"].(string)
		if oldest == "" || c.CreatedAt.Before(first) || (c.CreatedAt.Equal(first) && opened < firstOpened) {
			oldest, first, firstOpened = c.ID, c.CreatedAt, opened
		}
	}
	var out []*store.Record
	for _, m := range recs {
		in, _ := m.Fields["conversation"].(string)
		if in == id || (in == "" && id == oldest) {
			out = append(out, m)
		}
	}
	return out, nil
}

// message records one message in the current chat. The first thing the
// person says in a chat names it.
func (s *Service) message(fields map[string]any) (*store.Record, error) {
	id := s.Current()
	fields["conversation"] = id
	rec, err := s.Store.Create(MessageType, s.fields(MessageType, fields))
	if err != nil {
		return nil, err
	}
	if fields["role"] == "user" && id != "" {
		if c, err := s.Store.Get(ConversationType, id); err == nil {
			if title, _ := c.Fields["title"].(string); title == "" {
				content, _ := fields["content"].(string)
				s.Store.Update(ConversationType, id, map[string]any{"title": truncate(strings.TrimSpace(content), 60), "opened": now()})
			}
		}
	}
	return rec, nil
}

// Notice records something the page needs to tell the person, in the
// current chat, where every other problem is reported.
func (s *Service) Notice(text string) {
	s.message(map[string]any{"role": "error", "content": text})
}

// Say puts a message from the assistant in the current chat outside a
// turn: a step the person has to take that sameway learns of by itself,
// such as signing in to Tailscale.
func (s *Service) Say(text string) {
	s.message(map[string]any{"role": "assistant", "content": text})
}

// Clear deletes every message in the current chat. The canvas, and the
// other chats, are left alone.
func (s *Service) Clear() error {
	if err := s.Available(); err != nil {
		return err
	}
	Record(s.Store, "human", Change{Action: "cleared", Component: "conversation"})
	// The questions the assistant asked were part of the conversation; a
	// cleared one has no questions still waiting under it.
	for _, p := range s.Proposals() {
		s.Store.Update(ProposalType, p.ID, map[string]any{"state": "dismissed"})
	}
	msgs, err := s.Messages()
	if err != nil {
		return err
	}
	for _, m := range msgs {
		if err := s.Store.Delete(MessageType, m.ID); err != nil {
			return err
		}
	}
	return nil
}

// now is the moment as an opened stamp: fixed width, so that the order
// of the stamps as strings is the order of the moments, and never the
// same twice, since a clock can read the same for two moments in a row.
func now() string {
	stampMu.Lock()
	defer stampMu.Unlock()
	t := time.Now().UTC()
	if !t.After(lastStamp) {
		t = lastStamp.Add(time.Nanosecond)
	}
	lastStamp = t
	return t.Format("2006-01-02T15:04:05.000000000Z")
}

var (
	stampMu   sync.Mutex
	lastStamp time.Time
)
