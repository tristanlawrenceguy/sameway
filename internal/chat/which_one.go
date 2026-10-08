package chat

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Asked to "mark the invoice task done" with two invoice tasks, a small
// model ticked one of them, every time, and said nothing of the other. So
// a change to one record that the person's words fit no better than
// another's is stopped once, with both named, for the model to ask which;
// the same change sent again goes through, so a model that has asked, or
// knows better, is not held up. Words that speak of several (all, both,
// the urgent ones, tasks) are a change to several, and not stopped.

var whichAsked sync.Map // message id + record id -> true

// several are the words of a request about more than one thing.
var several = map[string]bool{"all": true, "both": true, "every": true, "each": true, "them": true, "those": true, "these": true, "ones": true, "everything": true}

// plain are words too common to tell records apart.
var plain = map[string]bool{"the": true, "and": true, "for": true, "with": true, "from": true, "mark": true, "set": true, "make": true, "done": true, "task": true, "note": true, "change": true, "move": true, "please": true, "can": true, "you": true, "my": true, "this": true, "that": true}

// whichOne is the refusal, or ok when the person's words point at rec.
func (s *Service) whichOne(t *schema.Type, rec *store.Record) (toolResult, bool) {
	msg, said := s.latestAsk()
	if msg == "" || s.actor() != "assistant" { // an agent from outside is not in the conversation
		return toolResult{}, true
	}
	asked := nameWords(said)
	for w := range asked {
		if several[w] || w == schema.Plural(t.Name) {
			return toolResult{}, true
		}
	}
	mine := nameWords(records.Title(s.Store, t, rec))
	shared := map[string]bool{}
	for w := range asked {
		if mine[w] && !plain[w] && len([]rune(w)) >= 3 {
			shared[w] = true
		}
	}
	if len(shared) == 0 {
		return toolResult{}, true
	}
	others, err := s.Store.List(t.Name, store.ListOptions{})
	if err != nil {
		return toolResult{}, true
	}
	for _, o := range others {
		if o.ID == rec.ID {
			continue
		}
		theirs := nameWords(records.Title(s.Store, t, o))
		fits := true
		for w := range shared {
			fits = fits && theirs[w]
		}
		apart := false // a word the person said that is in this title and not that one
		for w := range asked {
			apart = apart || mine[w] && !theirs[w] && !plain[w] && len([]rune(w)) >= 3
		}
		if !fits || apart {
			continue
		}
		key := msg + "/" + rec.ID
		if _, again := whichAsked.LoadOrStore(key, true); again {
			return toolResult{}, true
		}
		return fail("nothing was changed: the person's words, %q, fit %q no better than %q (%s %s). Ask them which one they mean; if you know it is this one, send the same change again.",
			strings.TrimSpace(said), records.Title(s.Store, t, rec), records.Title(s.Store, t, o), t.Name, o.ID), false
	}
	return toolResult{}, true
}

// latestAsk is the person's latest message: its id and its words.
func (s *Service) latestAsk() (id, words string) {
	msgs, _ := s.Store.List(records.MessageType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 10})
	for _, m := range msgs {
		if m.Fields["role"] == "user" {
			if time.Since(m.CreatedAt) > 15*time.Minute {
				return "", "" // not this turn's
			}
			return m.ID, fmt.Sprint(m.Fields["content"])
		}
	}
	return "", ""
}
