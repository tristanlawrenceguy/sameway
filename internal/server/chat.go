package server

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

//go:embed conversation.html
var conversationSrc string

var conversationTmpl = template.Must(template.New("conversation").Funcs(render.Funcs).Parse(conversationSrc))

// conversation is the rendered transcript and composer, plus the facts the
// page around it needs. The same value fills a chat block on the canvas and
// the standalone /chat page.
type conversation struct {
	Body     template.HTML
	Notice   template.HTML
	Activity template.HTML
	LatestID string
	// LastTurn and TurnEnd bound the most recent exchange: from the person's
	// message to the reply that closed it. Blocks touched inside that window
	// are what the assistant just did.
	LastTurn time.Time
	TurnEnd  time.Time
	Count    int
	// FocusID is the block the page is already showing in full, when it is
	// showing one. Its own Expand link then says it is the current page
	// rather than offering to go where you already are.
	FocusID string
	// Arrival is the order in which the blocks changed in the last turn
	// arrive on the page, by block id, so several changes are shown one
	// after another in the order they were made.
	Arrival map[string]int
}

type conversationView struct {
	Status    template.HTML
	Messages  []chatMessage
	Compose   template.HTML
	Send      template.HTML
	Clear     template.HTML
	ModelName string
	From      string
	Proposals []template.HTML
}

type chatMessage struct {
	ID   string
	HTML template.HTML
}

// conversation renders the transcript and composer once, for whichever
// surface is showing it.
func (s *Server) conversation(from string) (*conversation, error) {
	out := &conversation{}
	view := conversationView{From: from}
	if s.app.Chat.Provider == nil {
		problem := "No model is configured."
		if s.app.Chat.ProviderErr != nil {
			problem = s.app.Chat.ProviderErr.Error()
		}
		out.Notice = s.component("alert", map[string]any{"kind": "warning", "title": "No model connected",
			"message": problem + " To connect one, edit the llm part of workspace.yaml and start sameway again."})
	} else {
		view.ModelName = s.app.Chat.Provider.Name()
	}

	msgs, err := s.app.Store.List(chat.MessageType, store.ListOptions{OrderBy: "created_at"})
	if err != nil {
		return nil, err
	}
	for i, m := range msgs {
		if m.Fields["role"] == "user" {
			out.LastTurn = m.CreatedAt
		}
		out.TurnEnd = m.CreatedAt
		id := "msg-" + m.ID
		out.LatestID = id
		props := map[string]any{
			"role": m.Fields["role"], "content": m.Fields["content"], "id": id, "from": from,
			"time": m.CreatedAt.Local().Format("15:04"), "changes": s.undoable(m.Fields["changes"], i == len(msgs)-1),
		}
		if fileID, _ := m.Fields["file"].(string); fileID != "" {
			props["attachment"] = s.attachment(fileID)
		}
		view.Messages = append(view.Messages, chatMessage{ID: id, HTML: s.component("message", props)})
	}
	out.Count = len(msgs)
	view.Status = s.status(msgs)
	view.Proposals = s.proposals(from)
	view.Compose = s.component("textarea", map[string]any{"label": "Your message", "name": "message", "rows": 3, "required": true, "hint": "Ask for anything, or ask what something on the page is. Enter sends; Shift+Enter starts a new line."})
	view.Send = s.component("button", map[string]any{"label": "Send", "type": "submit"})
	view.Clear = s.component("button", map[string]any{"label": "Clear", "context": "conversation", "type": "submit", "variant": "quiet"})

	var body bytes.Buffer
	if err := conversationTmpl.Execute(&body, view); err != nil {
		return nil, err
	}
	out.Body = template.HTML(body.String())
	out.Activity = s.recentActivity(8, from)
	return out, nil
}

// attachment is how a message shows the file that came with it: its title
// as a link to its page, or just a word when the file has since gone.
func (s *Server) attachment(fileID string) map[string]any {
	rec, err := s.app.Store.Get(FileType, fileID)
	if err != nil {
		return map[string]any{"title": "a file that is no longer here"}
	}
	title, _ := rec.Fields["title"].(string)
	if title == "" {
		title = "a file"
	}
	return map[string]any{"title": title, "href": "/t/" + FileType + "/" + rec.ID}
}

// chatPage shows the conversation on its own page. It is always here, even
// when the canvas has no chat block, so a layout choice can never take the
// assistant away.
func (s *Server) chatPage(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Chat.Available(); err != nil {
		s.page(w, r, "Chat", s.component("alert", map[string]any{"kind": "danger", "title": "Chat is not available", "message": err.Error()}), pageOptions{})
		return
	}
	convo, err := s.conversation("/chat")
	if err != nil {
		s.fail(w, err)
		return
	}
	body, err := s.app.Registry.RenderSlot(chat.ComponentName, map[string]any{"layout": "bare"}, convo.Body)
	if err != nil {
		s.fail(w, err)
		return
	}
	opts := pageOptions{JSONURL: "/api/message"}
	if convo.LatestID != "" {
		opts.Focus, opts.FocusLabel = convo.LatestID, "Skip to latest message"
	}
	s.page(w, r, "Chat", template.HTML(string(convo.Notice)+string(body)+string(convo.Activity)), opts)
}

// backTo is where a conversation form returns to: the surface it was sent
// from, when that is one of ours, and the canvas otherwise. Only the paths
// that actually show a conversation are accepted, so the field cannot be
// used to bounce someone somewhere else.
func backTo(from string) string {
	if from == "/chat" {
		return from
	}
	for _, prefix := range []string{"/canvas/", "/c/"} {
		if id, ok := strings.CutPrefix(from, prefix); ok && id != "" && !strings.ContainsAny(id, "/?#") {
			return from
		}
	}
	return "/"
}

// status summarises the last turn for the live region.
func (s *Server) status(msgs []*store.Record) template.HTML {
	props := map[string]any{"id": "chat-status", "message": "Ready.", "state": "idle"}
	if len(msgs) > 0 {
		last := msgs[len(msgs)-1]
		switch last.Fields["role"] {
		case "error":
			props["state"], props["message"], props["live"] = "error", "The last request failed. See the message below.", "assertive"
		case "assistant":
			n := 0
			if changes, ok := last.Fields["changes"].([]any); ok {
				n = len(changes)
			}
			props["state"] = "done"
			switch n {
			case 0:
				props["message"] = "Assistant replied."
			case 1:
				props["message"] = "Assistant replied and made 1 change to the canvas."
			default:
				props["message"] = fmt.Sprintf("Assistant replied and made %d changes to the canvas.", n)
			}
		}
	}
	return s.component("status", props)
}

// chatSend handles the compose form, then returns to where it was sent from.
func (s *Server) chatSend(w http.ResponseWriter, r *http.Request) {
	// A file sent with the message is filed first, as its own record, and
	// goes to the model with the words. The composer is multipart for that;
	// a plain form still works for anything that posts without a file.
	fileID := ""
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		file, err := s.storeUpload(r)
		if err != nil && err != http.ErrMissingFile {
			s.fail(w, err)
			return
		}
		if file != nil {
			fileID = file.ID
		}
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	back := backTo(r.PostForm.Get("from"))
	// The tab the person typed on is the one the assistant builds on.
	canvas := strings.TrimPrefix(strings.TrimPrefix(back, "/c/"), "/")
	if !strings.HasPrefix(back, "/c/") {
		canvas = ""
	}
	rec, err := s.app.Chat.SendFile(r.Context(), canvas, r.PostForm.Get("message"), fileID)
	if rec == nil {
		// Nothing was recorded (empty message, or chat unavailable). The page
		// already explains the latter, so just show it again.
		if err != nil {
			log.Printf("chat: %v", err)
		}
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, back+"#msg-"+rec.ID, http.StatusSeeOther)
}

func (s *Server) chatClear(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Chat.Clear(); err != nil {
		s.fail(w, err)
		return
	}
	r.ParseForm()
	http.Redirect(w, r, backTo(r.PostForm.Get("from")), http.StatusSeeOther)
}
