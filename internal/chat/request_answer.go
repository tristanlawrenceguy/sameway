package chat

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A request is answered back to whoever asked: someone the owner let in
// hears it in their own conversation, and anyone, a visitor or another
// person's agent, reads what became of it at the request's own address,
// which only the one given it knows.

// RequestPath is the address of what became of a request.
func RequestPath(token string) string { return "/request/" + token }

func requestToken() string {
	b := make([]byte, 10)
	rand.Read(b)
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b))
}

// RequestGone is what a request's address says changed once what a yes
// made is taken away.
const RequestGone = "taken away"

// RequestState is what became of a request, in a word: waiting, yes or
// no.
func RequestState(p *store.Record) string {
	switch p.Fields["state"] {
	case "accepted":
		return "yes"
	case "dismissed":
		return "no"
	}
	return "waiting"
}

// RequestByToken is the request with that key, if there is one.
func (s *Service) RequestByToken(token string) *store.Record {
	if token == "" {
		return nil
	}
	recs, _ := s.Store.List(records.ProposalType, store.ListOptions{})
	for _, p := range recs {
		if p.Fields["by_action"] == ByRequest && p.Fields["token"] == token {
			return p
		}
	}
	return nil
}

// RequestAsker is who asked for a request, as Sameway knew them.
func RequestAsker(p *store.Record) (records.Visitor, bool) {
	raw, _ := p.Fields["asked_by"].(string)
	var v records.Visitor
	if raw == "" || json.Unmarshal([]byte(raw), &v) != nil {
		return v, false
	}
	return v, true
}

// tellAsker says the owner's answer to whoever asked, in their own
// conversation, when they have one: a visitor from the internet has its
// request's address instead.
func (s *Service) tellAsker(p *store.Record, yes bool) {
	v, ok := RequestAsker(p)
	if p.Fields["by_action"] == ByRequest && ok && v.Device == ByEmail {
		s.tellByMail(p, v, yes) // email_conversation.go: in their thread
		return
	}
	if p.Fields["by_action"] != ByRequest || !ok || v.Owner() {
		return
	}
	what, _ := p.Fields["asked"].(string)
	owner := s.Owner.Name
	if owner == "" {
		owner = "The owner"
	}
	said := owner + " said no to your request to " + what + ": nothing was made."
	if yes {
		said = owner + " said yes to your request to " + what + ": it is made."
	}
	if words, _ := p.Fields["answer"].(string); words != "" {
		said += "\n\n" + words
	}
	if s.Channel != nil && s.Channel(v, said) { // channel.go: where they asked, such as Telegram
		return
	}
	if v.Access != records.Public {
		s.For(v).Say(said)
	}
}
