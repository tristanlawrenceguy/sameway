package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Each person who may edit the workspace has their own conversations with
// the assistant: the owner's are the owner's, and Bob's are Bob's. For
// gives the service as one person has it, for one request: the same
// workspace, their own chats and turns, and the tools their access allows.
// A conversation says whose it is by the person's login; the owner's say
// nobody's, as every chat did before there were other people.

// For is the service as v has it. The owner has the service itself.
func (s *Service) For(v records.Visitor) *Service {
	if v.Owner() || v.Access == "" {
		return s
	}
	c := *s
	c.who, c.convo, c.current = v, "", ""
	return &c
}

// owner says whether the one this service speaks for may do everything.
func (s *Service) owner() bool { return s.who.Access == "" || s.who.Owner() }

// whose is the key a conversation carries for the one this service speaks
// for: "" for the owner, a login for anyone else.
func (s *Service) whose() string {
	if s.owner() {
		return ""
	}
	return strings.ToLower(s.who.Login)
}

// Whose is whose chats and turns this service has: "" for the owner, a
// login for anyone else.
func (s *Service) Whose() string { return s.whose() }

// IsOwner says whether the one this service speaks for may do everything.
func (s *Service) IsOwner() bool { return s.owner() }

// mine says whether a conversation is this person's.
func (s *Service) mine(c *store.Record) bool {
	p, _ := c.Fields["person"].(string)
	return strings.EqualFold(p, s.whose())
}

// OwnersAlone says whether a tool is the owner's alone (its Op says ForOwner:
// settings, updating the program, a browser on the machine, undoing, which
// can put back a setting or someone's access), for the test that
// the pages and the assistant agree on what is.
func OwnersAlone(tool string) bool {
	op, _ := OpFor(tool)
	return op.Access == ForOwner
}

// Tools are the tools the model is offered, for the one it speaks for.
func (s *Service) Tools() []llm.Tool {
	all := s.allTools()
	if s.owner() {
		return all
	}
	out := all[:0:0]
	for _, t := range all {
		if !OwnersAlone(t.Name) {
			out = append(out, t)
		}
	}
	return out
}

// refuseFor stops a tool the one this service speaks for may not use,
// whatever the model tried.
func (s *Service) refuseFor(call llm.ToolCall) (toolResult, bool) {
	if s.owner() || !OwnersAlone(call.Name) {
		return toolResult{}, false
	}
	return fail("%s is for the workspace's owner; say they can ask for it", call.Name), true
}

// whoPrompt tells the model who it is talking to, when that is not the
// owner.
func (s *Service) whoPrompt() string {
	if s.owner() {
		return ""
	}
	name := s.who.Who()
	return fmt.Sprintf("\n\nYou are talking with %s, whom the workspace's owner has let %s it from their own device. This is their own conversation; the owner does not see it. Settings, updating Sameway, reading pages in a browser and undoing are the owner's: if %s asks for one, say the owner can. What you ask them to agree to goes to the owner to answer.", name, s.who.Access, name)
}

// forYouPrompt tells the model what is for the one it is talking to: the
// records that point at them as a person, not yet done, so "what is for
// me?" has an answer.
func (s *Service) forYouPrompt() string {
	login := s.Owner.Login
	if !s.owner() {
		login = s.who.Login
	}
	me := s.PersonByEmail(strings.ToLower(login))
	if me == nil {
		return ""
	}
	var lines []string
	for _, t := range records.ContentTypes(s.Store) {
		for _, f := range t.Fields {
			if f.Type != "ref" || f.To != records.PersonType {
				continue
			}
			recs, _ := s.Store.List(t.Name, store.ListOptions{OrderBy: "updated_at", Desc: true, Limit: 50})
			for _, r := range recs {
				if r.Fields[f.Name] != me.ID || t.Done(r.Fields) || len(lines) >= 10 {
					continue
				}
				lines = append(lines, fmt.Sprintf("- %s: %s (/t/%s/%s)", t.Name, records.Name(s.Store, t, r), t.Name, r.ID))
			}
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n\nFor the person you are talking to, not yet done:\n" + strings.Join(lines, "\n")
}
