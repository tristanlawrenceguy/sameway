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
	case p != nil && (p.Fields["access"] == chat.View || p.Fields["access"] == chat.Edit):
		v.Access, _ = p.Fields["access"].(string)
	default:
		s.app.Chat.Knock(login, name, device)
		return ctx, "You have asked to open " + s.app.Workspace.Config.Name + ". It opens here once its owner says yes: reload this page then.\n", false
	}
	return chat.WithVisitor(ctx, v), "", true
}

// ownerOnly are the parts of the workspace that are its owner's alone:
// the conversation with the assistant and its questions (one shared chat,
// until each person has their own), the other workspaces on the machine,
// the model, the comfort settings, and a browser driven on the machine.
var ownerOnly = []string{
	"/chat", "/api/chat", "/api/look", "/proposal", "/workspaces", "/model", "/help/set", "/activity",
	"/t/message", "/api/message", "/t/conversation", "/api/conversation",
	"/t/proposal", "/api/proposal", "/t/activity", "/api/activity",
}

// allowed says whether a visitor may make this request, and when not,
// tells them so on a page of its own.
func (s *Server) allowed(w http.ResponseWriter, r *http.Request) bool {
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
	if why == "" && v.Access != chat.Edit && r.Method != http.MethodGet && r.Method != http.MethodHead {
		why = "You can look at this workspace but not change it. Its owner can let you edit."
	}
	if why == "" {
		return true
	}
	s.page(w, r, "Not yours to change", s.component("alert", map[string]any{"kind": "info", "message": why}), pageOptions{Status: http.StatusForbidden})
	return false
}

// conversationFor is the conversation as the one asking may see it: the
// owner's in full; for anyone else, a word that the assistant here is the
// owner's for now, and nothing of what was said to it.
func (s *Server) conversationFor(r *http.Request, from string) (*conversation, error) {
	convo, err := s.conversation(from)
	if err != nil || chat.VisitorOf(r.Context()).Owner() {
		return convo, err
	}
	convo.Body = template.HTML(`<p class="sw-muted">The assistant here is for the workspace's owner, for now.</p>`)
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
