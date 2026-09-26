package server

import (
	"context"
	"html/template"
	"net/http"
	"strings"

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

// ownerOnly are the parts of the workspace that are its owner's alone:
// the assistant's questions, every chat's raw records and the log of what
// was said, the other workspaces on the machine, the model, the comfort
// settings, and a browser driven on the machine. Each person's own chat is
// theirs (see chatFor).
var ownerOnly = []string{
	"/api/look", "/proposal", "/workspaces", "/model", "/help/set", "/activity",
	"/t/message", "/api/message", "/t/conversation", "/api/conversation",
	"/t/proposal", "/api/proposal", "/t/activity", "/api/activity",
}

// allowed says whether a visitor may make this request, and when not,
// tells them so on a page of its own.
func (s *Server) allowed(w http.ResponseWriter, r *http.Request) bool {
	if isPage(r) {
		s.seen(r, r.URL.Path)
	}
	v := chat.VisitorOf(r.Context())
	if v.Owner() {
		return true
	}
	why := ""
	for _, p := range ownerOnly {
		if r.URL.Path == p || strings.HasPrefix(r.URL.Path, p+"/") {
			why = "This part of the workspace is its owner's alone."
		}
	}
	// An import writes whatever its columns say, access included.
	if strings.HasSuffix(r.URL.Path, "/import") || strings.Contains(r.URL.Path, "/import/") {
		why = "This part of the workspace is its owner's alone."
	}
	if why == "" && v.Access != chat.Edit && v.Access != chat.Host && r.Method != http.MethodGet && r.Method != http.MethodHead {
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
	return s.conversationAboutFor(r, from, "", "")
}

func (s *Server) conversationAboutFor(r *http.Request, from, about, prompt string) (*conversation, error) {
	convo, err := s.conversationAbout(s.chatFor(r), from, about, prompt)
	if err != nil || chat.VisitorOf(r.Context()).Access != chat.View {
		return convo, err
	}
	convo.Body = template.HTML(`<p class="sw-muted">You can look around this workspace. The assistant is for the people who can change it.</p>`)
	convo.Notice, convo.Activity, convo.LatestID = "", "", ""
	return convo, nil
}

// keptFromVisitor refuses, to anyone but the owner, a write through the
// API that sets a field Sameway keeps, such as a person's access: the
// pages already refuse it, and the API must not be the way round.
func (s *Server) keptFromVisitor(w http.ResponseWriter, r *http.Request, fields map[string]any) bool {
	if chat.VisitorOf(r.Context()).Owner() {
		return false
	}
	t, ok := s.app.Types.Get(r.PathValue("type"))
	if !ok {
		return false
	}
	for _, f := range t.Fields {
		if _, sent := fields[f.Name]; sent && f.ReadOnly {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": f.Name + " is kept by Sameway and cannot be set here"})
			return true
		}
	}
	return false
}
