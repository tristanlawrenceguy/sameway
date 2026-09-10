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

//go:embed chat.html
var chatSrc string

var chatTmpl = template.Must(template.New("chat").Funcs(render.Funcs).Parse(chatSrc))

type chatView struct {
	Notice    template.HTML
	Status    template.HTML
	Messages  []chatMessage
	Blocks    []canvasBlock
	Compose   template.HTML
	Send      template.HTML
	Clear     template.HTML
	ModelName string
	Activity  []template.HTML
}

type chatMessage struct {
	ID   string
	HTML template.HTML
}

type canvasBlock struct {
	ID        string
	Component string
	Actor     string
	Changed   string
	HTML      template.HTML
	Badge     template.HTML
	Edit      template.HTML
	Remove    template.HTML
}

// chatPage is the home page: the conversation and the canvas side by side,
// with provenance on every block, change markers from the last turn, a
// live status, and the recent activity log.
func (s *Server) chatPage(w http.ResponseWriter, r *http.Request) {
	view := chatView{}
	if err := s.app.Chat.Available(); err != nil {
		view.Notice = s.component("alert", map[string]any{"kind": "danger", "title": "Chat is not available", "message": err.Error()})
	} else if s.app.Chat.Provider == nil {
		problem := "No model is configured."
		if s.app.Chat.ProviderErr != nil {
			problem = s.app.Chat.ProviderErr.Error()
		}
		view.Notice = s.component("alert", map[string]any{"kind": "warning", "title": "No model connected", "message": problem + " Edit the llm section of workspace.yaml and restart sameway serve."})
	} else {
		view.ModelName = s.app.Chat.Provider.Name()
	}
	focus := ""
	if s.app.Chat.Available() == nil {
		msgs, err := s.app.Store.List(chat.MessageType, store.ListOptions{OrderBy: "created_at"})
		if err != nil {
			s.fail(w, err)
			return
		}
		var lastTurn time.Time
		for _, m := range msgs {
			if m.Fields["role"] == "user" {
				lastTurn = m.CreatedAt
			}
			id := "msg-" + m.ID
			focus = id
			view.Messages = append(view.Messages, chatMessage{ID: id, HTML: s.component("message", map[string]any{
				"role": m.Fields["role"], "content": m.Fields["content"], "id": id,
				"time": m.CreatedAt.Local().Format("15:04"), "changes": m.Fields["changes"],
			})})
		}
		view.Status = s.status(msgs)
		blocks, err := s.app.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
		if err != nil {
			s.fail(w, err)
			return
		}
		for _, b := range blocks {
			view.Blocks = append(view.Blocks, s.canvasBlock(b, lastTurn))
		}
		view.Activity = s.recentActivity(6)
	}
	view.Compose = s.component("textarea", map[string]any{"label": "Your message", "name": "message", "rows": 3, "required": true, "hint": "Ask for content or a component. Send with the button."})
	view.Send = s.component("button", map[string]any{"label": "Send", "type": "submit"})
	view.Clear = s.component("button", map[string]any{"label": "Clear conversation", "type": "submit", "variant": "secondary"})

	var body bytes.Buffer
	if err := chatTmpl.Execute(&body, view); err != nil {
		s.fail(w, err)
		return
	}
	opts := pageOptions{JSONURL: "/api/message"}
	if focus != "" {
		opts.Focus, opts.FocusLabel = focus, "Skip to latest message"
	}
	s.page(w, r, "Chat", template.HTML(body.String()), opts)
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

// canvasBlock renders one block with its provenance badge, change marker,
// and the actions a person can take on it.
func (s *Server) canvasBlock(b *store.Record, lastTurn time.Time) canvasBlock {
	name, _ := b.Fields["component"].(string)
	props, _ := b.Fields["props"].(map[string]any)
	actor, _ := b.Fields["actor"].(string)
	if actor == "" {
		actor = "assistant"
	}
	createdBy, _ := b.Fields["created_by"].(string)
	if createdBy == "" {
		createdBy = actor
	}
	who := map[string]string{"human": "you", "assistant": "assistant"}
	edited := actor != createdBy || b.UpdatedAt.Sub(b.CreatedAt) > time.Second
	label := "Added by " + who[createdBy]
	if edited {
		label = "Added by " + who[createdBy] + ", edited by " + who[actor]
	}
	label += " · " + b.UpdatedAt.Local().Format("15:04")
	changed := ""
	if !lastTurn.IsZero() {
		if !b.CreatedAt.Before(lastTurn) {
			changed = "added"
		} else if !b.UpdatedAt.Before(lastTurn) {
			changed = "updated"
		}
	}
	return canvasBlock{
		ID: b.ID, Component: name, Actor: actor, Changed: changed,
		HTML:   s.component(name, props),
		Badge:  s.component("badge", map[string]any{"label": label, "tone": actor}),
		Edit:   s.component("link", map[string]any{"href": "/t/block/" + b.ID + "/edit", "label": "Edit " + name}),
		Remove: s.component("button", map[string]any{"label": "Remove " + name, "type": "submit", "variant": "secondary"}),
	}
}

// chatSend handles the compose form, then redirects to the newest message.
func (s *Server) chatSend(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	rec, err := s.app.Chat.Send(r.Context(), r.PostForm.Get("message"))
	if rec == nil {
		// Nothing was recorded (empty message, or chat unavailable). The page
		// already explains the latter, so just show it again.
		if err != nil {
			log.Printf("chat: %v", err)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/#msg-"+rec.ID, http.StatusSeeOther)
}

func (s *Server) chatClear(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Chat.Clear(); err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// canvasDelete is a person removing a block; it is logged as a human action.
func (s *Server) canvasDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rec, err := s.app.Store.Get(chat.BlockType, id)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.app.Store.Delete(chat.BlockType, id); err != nil {
		s.fail(w, err)
		return
	}
	name, _ := rec.Fields["component"].(string)
	props, _ := rec.Fields["props"].(map[string]any)
	chat.Record(s.app.Store, "human", chat.Change{Action: "removed", Component: name, ID: id, Detail: chat.Summarise(name, props)})
	http.Redirect(w, r, "/#canvas", http.StatusSeeOther)
}
