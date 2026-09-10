package server

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
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
	LastTurn time.Time
	Count    int
}

type conversationView struct {
	Status    template.HTML
	Messages  []chatMessage
	Compose   template.HTML
	Send      template.HTML
	Clear     template.HTML
	ModelName string
	From      string
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
			"message": problem + " Edit the llm section of workspace.yaml and restart sameway serve."})
	} else {
		view.ModelName = s.app.Chat.Provider.Name()
	}

	msgs, err := s.app.Store.List(chat.MessageType, store.ListOptions{OrderBy: "created_at"})
	if err != nil {
		return nil, err
	}
	for _, m := range msgs {
		if m.Fields["role"] == "user" {
			out.LastTurn = m.CreatedAt
		}
		id := "msg-" + m.ID
		out.LatestID = id
		view.Messages = append(view.Messages, chatMessage{ID: id, HTML: s.component("message", map[string]any{
			"role": m.Fields["role"], "content": m.Fields["content"], "id": id,
			"time": m.CreatedAt.Local().Format("15:04"), "changes": m.Fields["changes"],
		})})
	}
	out.Count = len(msgs)
	view.Status = s.status(msgs)
	view.Compose = s.component("textarea", map[string]any{"label": "Your message", "name": "message", "rows": 3, "required": true})
	view.Send = s.component("button", map[string]any{"label": "Send", "type": "submit"})
	view.Clear = s.component("button", map[string]any{"label": "Clear", "context": "conversation", "type": "submit", "variant": "quiet"})

	var body bytes.Buffer
	if err := conversationTmpl.Execute(&body, view); err != nil {
		return nil, err
	}
	out.Body = template.HTML(body.String())
	out.Activity = s.recentActivity(8)
	return out, nil
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
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	back := "/"
	if from := r.PostForm.Get("from"); from == "/chat" {
		back = from
	}
	rec, err := s.app.Chat.Send(r.Context(), r.PostForm.Get("message"))
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
	back := "/"
	if from := r.PostForm.Get("from"); from == "/chat" {
		back = from
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}
