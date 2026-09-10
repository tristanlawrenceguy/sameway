package server

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/http"

	"github.com/sameway-dev/sameway/internal/chat"
	"github.com/sameway-dev/sameway/internal/render"
	"github.com/sameway-dev/sameway/internal/store"
)

//go:embed chat.html
var chatSrc string

var chatTmpl = template.Must(template.New("chat").Funcs(render.Funcs).Parse(chatSrc))

type chatView struct {
	Notice    template.HTML
	Messages  []template.HTML
	Blocks    []canvasBlock
	Compose   template.HTML
	Send      template.HTML
	Clear     template.HTML
	ModelName string
}

type canvasBlock struct {
	ID        string
	Component string
	HTML      template.HTML
	Remove    template.HTML
}

// chatPage is the home page: the conversation and the canvas side by side.
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
		for _, m := range msgs {
			id := "msg-" + m.ID
			focus = id
			view.Messages = append(view.Messages, s.component("message", map[string]any{
				"role": m.Fields["role"], "content": m.Fields["content"], "id": id, "time": m.CreatedAt.Local().Format("15:04"),
			}))
		}
		blocks, err := s.app.Store.List(chat.BlockType, store.ListOptions{OrderBy: "position"})
		if err != nil {
			s.fail(w, err)
			return
		}
		for _, b := range blocks {
			name, _ := b.Fields["component"].(string)
			props, _ := b.Fields["props"].(map[string]any)
			view.Blocks = append(view.Blocks, canvasBlock{
				ID: b.ID, Component: name, HTML: s.component(name, props),
				Remove: s.component("button", map[string]any{"label": "Remove " + name, "type": "submit", "variant": "secondary"}),
			})
		}
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

// chatSend handles the compose form, then redirects to the newest message.
func (s *Server) chatSend(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	rec, err := s.app.Chat.Send(r.Context(), r.PostForm.Get("message"))
	if rec == nil {
		if err != nil {
			s.fail(w, err)
			return
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

func (s *Server) canvasDelete(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Store.Delete(chat.BlockType, r.PathValue("id")); err != nil {
		s.fail(w, err)
		return
	}
	http.Redirect(w, r, "/#canvas", http.StatusSeeOther)
}
