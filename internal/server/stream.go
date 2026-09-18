package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A turn as it happens. The composer is a form and posts to /chat, where
// the whole turn runs and the page comes back; with scripts it posts the
// same form here instead and reads the turn as server-sent events: the
// person's message as recorded, the reply's words as they come, each tool
// as it starts, each block as it lands, rendered here so the page can put
// it where it belongs, and the reply as recorded. Nothing here exists for
// a browser without scripts; /chat does the same work in one go.

// chatInput reads the composer's form, with the file it may carry.
func (s *Server) chatInput(r *http.Request) (canvas, text, fileID, back string, err error) {
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		file, uerr := s.storeUpload(r)
		if uerr != nil && uerr != http.ErrMissingFile {
			return "", "", "", "", uerr
		}
		if file != nil {
			fileID = file.ID
		}
	}
	if err := r.ParseForm(); err != nil {
		return "", "", "", "", err
	}
	back = backTo(r.PostForm.Get("from"))
	canvas = strings.TrimPrefix(strings.TrimPrefix(back, "/c/"), "/")
	if !strings.HasPrefix(back, "/c/") {
		canvas = ""
	}
	return canvas, r.PostForm.Get("message"), fileID, back, nil
}

func (s *Server) chatStream(w http.ResponseWriter, r *http.Request) {
	canvas, text, fileID, back, err := s.chatInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not possible here", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	started := time.Now()
	// Events come from the turn and, when the tools run elsewhere, from
	// the watch on the log: one writer at a time.
	var mu sync.Mutex
	send := func(event string, data map[string]any) {
		body, _ := json.Marshal(data)
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, body)
		flusher.Flush()
	}
	// The turn runs to its end even when the page that asked for it goes
	// away: a tab closed mid-turn must not leave a change half made and an
	// error in the log where the reply should be. The person can stop it,
	// though, from the page, by the id the first event carries.
	turn, ctx, over := s.turns.start(context.WithoutCancel(r.Context()))
	defer over()
	rec, err := s.app.Chat.SendLive(ctx, canvas, text, fileID, func(e chat.Event) {
		switch e.Kind {
		case "said":
			send("said", map[string]any{"id": e.ID, "turn": turn, "html": s.messageHTML(e.ID, back, false)})
		case "delta", "text":
			send(e.Kind, map[string]any{"text": e.Text})
		case "tool":
			send("tool", map[string]any{"tool": e.Tool, "label": e.Label, "early": e.Early})
		case "change":
			data := map[string]any{"action": e.Change.Action, "component": e.Change.Component, "id": e.Change.ID, "detail": e.Change.Detail}
			if blk, region, html := s.liveBlock(e.Change, started); html != "" {
				data["block"], data["region"], data["html"] = blk, region, html
			}
			send("change", data)
		case "done":
			send("done", map[string]any{"id": e.ID, "html": s.messageHTML(e.ID, back, true), "status": string(s.statusFor(e.ID)), "activity": string(s.recentActivity(8, back))})
		case "error":
			send("error", map[string]any{"id": e.ID, "text": e.Text, "html": s.messageHTML(e.ID, back, true), "status": string(s.statusFor(e.ID)), "activity": string(s.recentActivity(8, back))})
		}
	})
	if rec == nil && err != nil {
		log.Printf("chat: %v", err)
		send("error", map[string]any{"text": err.Error()})
	}
}

// turns are the live turns under way, each with the way to stop it.
type turns struct {
	mu     sync.Mutex
	n      int
	cancel map[string]context.CancelFunc
}

// start begins a turn: its id, the context it runs under, and what to
// call when it is over.
func (t *turns) start(parent context.Context) (id string, ctx context.Context, over func()) {
	ctx, cancel := context.WithCancel(parent)
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel == nil {
		t.cancel = map[string]context.CancelFunc{}
	}
	t.n++
	id = fmt.Sprintf("turn-%d", t.n)
	t.cancel[id] = cancel
	return id, ctx, func() {
		cancel()
		t.mu.Lock()
		delete(t.cancel, id)
		t.mu.Unlock()
	}
}

// stop ends the turn named, or every turn under way when none is.
func (t *turns) stop(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, cancel := range t.cancel {
		if id == "" || k == id {
			cancel()
		}
	}
}

// chatStop is the Stop control: the turn ends where it is, its reply
// says so, and what it did stays. The stream that runs the turn tells
// the page the rest.
func (s *Server) chatStop(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	s.turns.stop(r.PostForm.Get("turn"))
	w.WriteHeader(http.StatusNoContent)
}

// liveBlock is a changed block rendered as it now is, or nothing when the
// change was not to a block that is still there. A removed block is still
// a change the page hears about; it just carries no HTML.
func (s *Server) liveBlock(c *chat.Change, started time.Time) (id, region, html string) {
	if c.ID == "" || c.Action == "removed" {
		return c.ID, "", ""
	}
	blk, err := s.app.Store.Get(chat.BlockType, c.ID)
	if err != nil {
		return c.ID, "", ""
	}
	region, _ = blk.Fields["region"].(string)
	if region == "" {
		region = "main"
	}
	convo := &conversation{LastTurn: started, TurnEnd: time.Now(), Arrival: map[string]int{blk.ID: 1}}
	return blk.ID, region, s.blockItem(blk, convo)
}

// messageHTML is one message as the page shows it.
func (s *Server) messageHTML(id, from string, last bool) string {
	m, err := s.app.Store.Get(chat.MessageType, id)
	if err != nil {
		return ""
	}
	props := map[string]any{
		"role": m.Fields["role"], "content": m.Fields["content"], "id": "msg-" + m.ID, "from": from,
		"time": m.CreatedAt.Local().Format("15:04"), "changes": s.undoable(m.Fields["changes"], last),
	}
	if fileID, _ := m.Fields["file"].(string); fileID != "" {
		props["attachment"] = s.attachment(fileID)
	}
	return string(s.component("message", props))
}

// statusFor is the status line after a reply, as the page shows it.
func (s *Server) statusFor(id string) string {
	m, err := s.app.Store.Get(chat.MessageType, id)
	if err != nil {
		return ""
	}
	return string(s.status([]*store.Record{m}))
}
