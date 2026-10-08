package server

import (
	"errors"
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Several chats. A person can start a new chat, go back to an earlier one,
// or delete one, from a menu at the top of the chat. Where the chat sits
// and how wide is the assistant's to arrange, as for any block: a person
// asks. Popping it out opens
// /chat in a small window of its own.

// chatItem is one chat in the menu.
type chatItem struct {
	ID      string
	Title   string
	When    string
	Current bool
}

// chats lists every chat for the menu and names the current one.
func (s *Server) chats(svc *chat.Service) (items []chatItem, current string) {
	id := svc.Current()
	for _, c := range svc.Conversations() {
		title := svc.Title(c)
		item := chatItem{ID: c.ID, Title: title, When: when.Date(c.CreatedAt, time.Now()), Current: c.ID == id}
		if item.Current {
			current = title
		}
		items = append(items, item)
	}
	return items, current
}

func (s *Server) chatNew(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if _, err := s.chatFor(r).NewChat(); err != nil {
		s.failed(w, r, "No new chat", err, "/")
		return
	}
	http.Redirect(w, r, backTo(r.PostForm.Get("from")), http.StatusSeeOther)
}

func (s *Server) chatOpen(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if err := s.chatFor(r).OpenChat(r.PostForm.Get("id")); err != nil {
		s.failed(w, r, "Chat not opened", errors.New("that chat is not there any more"), "/")
		return
	}
	http.Redirect(w, r, backTo(r.PostForm.Get("from")), http.StatusSeeOther)
}

func (s *Server) chatDelete(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if err := s.chatFor(r).DeleteChat(r.PostForm.Get("id")); err != nil {
		s.failed(w, r, "Chat not deleted", err, "/")
		return
	}
	// Deleting a chat is a press away from a mistake: it can be put back.
	s.tellAt(w, r, outcome{Title: "Chat deleted", Undo: s.lastAbout("conversation")}, backTo(r.PostForm.Get("from")))
}

// lastAbout is the newest log entry about a kind of thing, so the outcome
// of an action can offer to take it back.
func (s *Server) lastAbout(target string) string {
	recent, _ := s.app.Store.List(chat.ActivityType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 5})
	for _, a := range recent {
		if a.Fields["target"] == target {
			return a.ID
		}
	}
	return ""
}
