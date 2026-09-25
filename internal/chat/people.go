package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Each person who may edit the workspace has their own conversations with
// the assistant: the owner's are the owner's, and Bob's are Bob's. For
// gives the service as one person has it, for one request: the same
// workspace, their own chats and turns, and the tools their access allows.
// A conversation says whose it is by the person's login; the owner's say
// nobody's, as every chat did before there were other people.

// For is the service as v has it. The owner has the service itself.
func (s *Service) For(v Visitor) *Service {
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

// ownerTools are what only the owner may have the assistant do: change
// settings, update the program, drive a browser on the machine, and undo
// (which can put back a setting or someone's access).
var ownerTools = map[string]bool{"set_setting": true, "update_sameway": true, "look_at_page": true, "undo_change": true}

// Tools are the tools the model is offered, for the one it speaks for.
func (s *Service) Tools() []llm.Tool {
	all := s.allTools()
	if s.owner() {
		return all
	}
	out := all[:0:0]
	for _, t := range all {
		if !ownerTools[t.Name] {
			out = append(out, t)
		}
	}
	return out
}

// refuseFor stops a tool the one this service speaks for may not use,
// whatever the model tried.
func (s *Service) refuseFor(call llm.ToolCall) (toolResult, bool) {
	if s.owner() || !ownerTools[call.Name] {
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

// PersonColour is the colour someone's changes are shown in, 1 to 6, from
// their login: the same person has the same colour on every computer.
func PersonColour(login string) int {
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return 0
	}
	h := uint32(2166136261)
	for i := 0; i < len(login); i++ {
		h = (h ^ uint32(login[i])) * 16777619
	}
	return int(h%6) + 1
}
