package chat

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Who else may open this workspace, from their own devices over Tailscale,
// is a person record's access: view or edit, matched by the email they
// sign in to Tailscale with. Sameway keeps that field. It changes only
// here, and giving access is always put to the owner first, so nobody can
// let themselves in or raise their own level.

// PersonType is the content type people are kept in.
const PersonType = "person"

var letInTool = llm.Tool{
	Name:        "let_in",
	Description: "Give someone access to this workspace from their own devices over Tailscale, or take it away, when the owner asks: \"let Bob edit\", \"Carol can look\", \"stop Bob\". They are matched by the email they sign in to Tailscale with, and reach the workspace once the owner shares this machine with them in Tailscale (or they are on the same tailnet). view reads only; edit changes content and the canvas and presses buttons; host is edit, and their own computer keeps a full copy of the workspace in step with this one (for when they host it too, with their own assistant); none takes access away. Giving access is put to the owner as a question for you, and nothing changes until they say yes; taking it away happens at once.",
	Schema: obj(map[string]any{
		"email":  map[string]any{"type": "string"},
		"name":   map[string]any{"type": "string"},
		"access": map[string]any{"type": "string", "enum": []string{View, Edit, Host, "none"}},
	}, "email", "access"),
}

// accessTools offers let_in where there are people to let in.
func (s *Service) accessTools() []llm.Tool {
	if _, ok := s.Store.Types().Get(PersonType); !ok {
		return nil
	}
	return []llm.Tool{letInTool}
}

type letInArgs struct {
	Email  string `json:"email"`
	Name   string `json:"name"`
	Access string `json:"access"`
}

// letInCall is the assistant's call: giving access is asked, taking it
// away is done.
func (s *Service) letInCall(raw json.RawMessage) toolResult {
	var a letInArgs
	json.Unmarshal(raw, &a)
	a.Email = strings.ToLower(strings.TrimSpace(a.Email))
	if !strings.Contains(a.Email, "@") {
		return fail("let_in needs the email they sign in to Tailscale with, such as bob@example.com")
	}
	switch a.Access {
	case "none":
		if !s.owner() {
			return fail("only the workspace's owner can take someone's access away")
		}
		return s.letIn(a)
	case View, Edit, Host:
		q := letInQuestion(a)
		return s.ask(q, map[string]any{"tool": "let_in", "email": a.Email, "name": a.Name, "access": a.Access})
	}
	return fail("access is view, edit or none, not %q", a.Access)
}

func letInQuestion(a letInArgs) question {
	who := a.Email
	if a.Name != "" {
		who = a.Name + " (" + a.Email + ")"
	}
	can := "read everything here except your conversations with the assistant"
	ask, yes := "Let "+who+" look at this workspace?", "Yes, let them look"
	switch a.Access {
	case Edit:
		can = "read and change what is here (notes, records, the canvas) and press its buttons, but not its settings or your conversations with the assistant"
		ask, yes = "Let "+who+" edit this workspace?", "Yes, let them edit"
	case Host:
		can = "keep a full copy of it on their own computer, in step with this one, and change anything in it, including who else may come in. Your conversations with the assistant stay on this computer. Only say yes to someone you trust with all of it"
		ask, yes = "Let "+who+" host this workspace too?", "Yes, let them host it"
	}
	return question{ask,
		fmt.Sprintf("They would open it from their own devices, signed in to Tailscale as %s, and could %s. You can take it back at any time.", a.Email, can),
		yes, "No"}
}

// letIn sets a person's access, making the person when there is none with
// that email yet.
func (s *Service) letIn(a letInArgs) toolResult {
	a.Email = strings.ToLower(strings.TrimSpace(a.Email))
	access := a.Access
	if access == "none" {
		access = ""
	}
	p := s.PersonByEmail(a.Email)
	if p == nil {
		if access == "" {
			return toolResult{text: a.Email + " had no access"}
		}
		name := a.Name
		if name == "" {
			name, _, _ = strings.Cut(a.Email, "@")
		}
		rec, err := s.Store.Create(PersonType, map[string]any{"name": name, "email": a.Email, "access": access})
		if err != nil {
			return fail("could not add them: %v", err)
		}
		s.Say(s.shareWords(name, a.Email))
		return toolResult{text: name + " can now " + verb(access), change: &Change{Action: "let in", Component: PersonType, ID: rec.ID, Detail: name + " to " + access}}
	}
	before := p.Fields["access"]
	if _, err := s.Store.Update(PersonType, p.ID, map[string]any{"access": access}); err != nil {
		return fail("could not change their access: %v", err)
	}
	name, _ := p.Fields["name"].(string)
	// Someone who had no access has not been shared this machine yet.
	if access != "" && (before == nil || before == "") {
		s.Say(s.shareWords(name, a.Email))
	}
	action, detail := "let in", name+" to "+access
	if access == "" {
		action, detail = "took access from", name
	}
	return toolResult{text: name + " can now " + verb(access), change: &Change{Action: action, Component: PersonType, ID: p.ID, Detail: detail, Before: map[string]any{"access": before}}}
}

func verb(access string) string {
	switch access {
	case View:
		return "look at this workspace"
	case Edit:
		return "edit this workspace"
	case Host:
		return "host this workspace too"
	}
	return "no longer open this workspace"
}

// PersonByEmail is the person who signs in with this email, if any.
func (s *Service) PersonByEmail(email string) *store.Record {
	if _, ok := s.Store.Types().Get(PersonType); !ok || email == "" {
		return nil
	}
	people, err := s.Store.List(PersonType, store.ListOptions{})
	if err != nil {
		return nil
	}
	for _, p := range people {
		if e, _ := p.Fields["email"].(string); strings.EqualFold(strings.TrimSpace(e), email) {
			return p
		}
	}
	return nil
}

// Knock asks the owner, once, whether someone who reached the workspace
// over Tailscale without access may look, and says whether that was new.
func (s *Service) Knock(login, name, device string) bool {
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return false
	}
	for _, p := range s.Proposals() {
		if act, _ := p.Fields["action"].(map[string]any); act != nil && act["tool"] == "let_in" && act["email"] == login {
			return false
		}
	}
	a := letInArgs{Email: login, Name: name, Access: View}
	q := letInQuestion(a)
	if device != "" {
		q.detail = fmt.Sprintf("They tried to open it just now, from %s. ", device) + q.detail
	}
	s.ask(q, map[string]any{"tool": "let_in", "email": a.Email, "name": a.Name, "access": a.Access})
	return true
}

// shareWords tells the owner how the person they just let in reaches the
// workspace: Tailscale has to let them reach this machine, which is the
// owner's to do in Tailscale, not here.
func (s *Service) shareWords(name, email string) string {
	machine := strings.TrimSpace(s.setting("tailnet.name"))
	if machine == "" {
		return fmt.Sprintf("%s can come in once this workspace is on your Tailscale network: ask me to put it on your phone, and I will walk you through it.", name)
	}
	return fmt.Sprintf("One step is yours in Tailscale, if %s is not on your tailnet already: share this computer (%s) with %s from https://login.tailscale.com/admin/machines (its menu, then Share), and send them the link. Once they accept it, they open the same address you do.", name, machine, email)
}
