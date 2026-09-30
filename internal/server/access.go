package server

import (
	"context"
	"html/template"
	"net/http"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// Other people open the workspace over Tailscale, which says who they are;
// what they may do is their person record's access (see chat/access.go).
// A request from the machine itself carries no visitor and is the owner's.

// Admit decides who gets in over the tailnet: the owner's own devices, and
// anyone whose Tailscale login is the email of a person with access. Anyone
// else is told they have asked, and the owner is asked in the chat.
func (s *Server) Admit(ctx context.Context, login, name, device string, owner bool) (context.Context, string, bool) {
	v := chat.Visitor{Login: login, Name: name, Device: device}
	p := s.app.Chat.PersonByEmail(login)
	if p != nil {
		v.Person = p.ID
		if n, _ := p.Fields["name"].(string); n != "" {
			v.Name = n
		}
	}
	switch {
	case owner:
		v.Access = chat.Owner
	case p != nil && (p.Fields["access"] == chat.View || p.Fields["access"] == chat.Edit || p.Fields["access"] == chat.Host):
		v.Access, _ = p.Fields["access"].(string)
	default:
		// The owner hears of it where they are, the way a reminder rings.
		if s.app.Chat.Knock(login, name, device) && s.notify != nil {
			who := login
			if name != "" {
				who = name + " (" + login + ")"
			}
			go s.notify("Someone wants to open "+s.app.Workspace.Config.Name, who+" asked from "+device+". Answer in the chat.", s.linkTo("/"))
		}
		return ctx, "You have asked to open " + s.app.Workspace.Config.Name + ". It opens here once its owner says yes: reload this page then.\n", false
	}
	return chat.WithVisitor(ctx, v), "", true
}

// allowed says whether a visitor may make this request, and when not,
// tells them so on a page of its own.
func (s *Server) allowed(w http.ResponseWriter, r *http.Request) bool {
	// A page fetching itself to follow a change is not its person moving
	// about: its live connection says whether they are here (presence.go).
	if isPage(r) && !isPublic(r) && r.Header.Get("X-Requested-With") != "sameway-live" {
		s.seen(r, r.URL.Path)
	}
	v := chat.VisitorOf(r.Context())
	if v.Owner() {
		return true
	}
	why := ""
	if s.ownerOnlyRequest(r) { // see access_routes.go
		why = "This part of the workspace is its owner's alone."
	}
	// Saying they have caught up changes nothing but their own notice.
	if why == "" && v.Access != chat.Edit && v.Access != chat.Host && r.Method != http.MethodGet && r.Method != http.MethodHead && r.URL.Path != "/since/seen" {
		why = "You can look at this workspace but not change it. Its owner can let you edit."
	}
	if why == "" {
		return true
	}
	s.page(w, r, "Not yours to change", s.component("alert", map[string]any{"kind": "info", "message": why}), pageOptions{Status: http.StatusForbidden})
	return false
}

// chatFor is the assistant as the one asking has it: their own chats and
// turns, and the tools their access allows. See chat/people.go.
func (s *Server) chatFor(r *http.Request) *chat.Service {
	return s.app.Chat.For(chat.VisitorOf(r.Context()))
}

// conversationFor is the conversation of the one asking. Someone who may
// only look has none: the assistant changes things, so it is for those
// who may.
func (s *Server) conversationFor(r *http.Request, from string) (*conversation, error) {
	// ?prompt= puts words in the box wherever the conversation is, so a
	// thing to ask offered on a page is ready to send on that page.
	return s.conversationAboutFor(r, from, "", r.URL.Query().Get("prompt"))
}

func (s *Server) conversationAboutFor(r *http.Request, from, about, prompt string) (*conversation, error) {
	convo, err := s.conversationAbout(s.chatFor(r), from, about, prompt)
	if convo != nil {
		convo.Path, convo.Query = r.URL.Path, r.URL.Query()
	}
	if a := chat.VisitorOf(r.Context()).Access; err != nil || a != chat.View && a != chat.Public {
		return convo, err
	}
	convo.Body = template.HTML(`<p class="sw-muted">You can look around this workspace. The assistant is for the people who can change it.</p>`)
	convo.Notice, convo.Activity, convo.LatestID, convo.LookOnly = "", "", "", true
	return convo, nil
}
