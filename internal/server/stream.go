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
//
// A page opened while a turn runs, because the person went elsewhere
// after asking, carries the turn's id and follows it from /chat/live.

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
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming is not possible here", http.StatusNotImplemented)
		return
	}
	// The turn runs to its end even when the page that asked for it goes
	// away: a tab closed mid-turn must not leave a change half made and an
	// error in the log where the reply should be. The person can stop it,
	// though, from the page, by the id the first event carries, and any
	// page opened meanwhile can follow it, from /chat/live.
	t, ctx := s.turns.start(context.WithoutCancel(r.Context()))
	go func() {
		defer s.turns.end(t)
		rec, err := s.app.Chat.SendLive(ctx, canvas, text, fileID, t.add)
		if rec == nil && err != nil {
			log.Printf("chat: %v", err)
			t.add(chat.Event{Kind: "error", Text: err.Error()})
		}
		s.tellDone(t, rec, err, back)
	}()
	s.follow(w, r, t, back)
}

// chatLive is a turn under way, told from its start to its end, for a
// page opened while it runs: the one the person went to after asking, or
// the one they came back to. No content when no turn is running.
func (s *Server) chatLive(w http.ResponseWriter, r *http.Request) {
	t := s.turns.find(r.URL.Query().Get("turn"))
	if t == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming is not possible here", http.StatusNotImplemented)
		return
	}
	s.follow(w, r, t, backTo(r.URL.Query().Get("from")))
}

// tellDone tells the person a turn is over when no page heard it end:
// the tab was closed, or every page on it went away. A page that is
// following says so itself (19-live-join.js), so the news comes once.
// It goes the way a ringing reminder does, through notify.
func (s *Server) tellDone(t *liveTurn, rec *store.Record, err error, back string) {
	if s.notify == nil || t.followed() {
		return
	}
	title, text, path := "Assistant replied", "", back
	if rec != nil {
		text, _ = rec.Fields["content"].(string)
		path = back + "#msg-" + rec.ID
		if rec.Fields["role"] == "error" {
			title = "The assistant could not finish"
		}
	} else if err != nil {
		title, text = "The assistant could not finish", err.Error()
	}
	if r := []rune(strings.Join(strings.Fields(text), " ")); len(r) > 140 {
		text = string(r[:139]) + "…"
	}
	go s.notify(title, text, s.linkTo(path))
}

// follow writes a turn's events as server-sent events, those so far and
// then each as it comes, until the turn is over or the page goes away.
// Each is rendered for the page that listens, so its controls come back
// to that page.
func (s *Server) follow(w http.ResponseWriter, r *http.Request, t *liveTurn, back string) {
	flusher := w.(http.Flusher)
	t.listen(1)
	defer t.listen(-1)
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	send := func(event string, data map[string]any) {
		body, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, body)
		flusher.Flush()
	}
	for i := 0; ; {
		events, over, changed := t.since(i)
		i += len(events)
		for _, e := range events {
			switch e.Kind {
			case "said":
				send("said", map[string]any{"id": e.ID, "turn": t.id, "html": s.messageHTML(e.ID, back, false)})
			case "delta", "text":
				send(e.Kind, map[string]any{"text": e.Text})
			case "tool":
				send("tool", map[string]any{"tool": e.Tool, "label": e.Label, "early": e.Early})
			case "change":
				data := map[string]any{"action": e.Change.Action, "component": e.Change.Component, "id": e.Change.ID, "detail": e.Change.Detail}
				if blk, region, html := s.liveBlock(e.Change, t.started); html != "" {
					data["block"], data["region"], data["html"] = blk, region, html
				}
				send("change", data)
			case "done":
				send("done", map[string]any{"id": e.ID, "html": s.messageHTML(e.ID, back, true), "status": string(s.statusFor(e.ID)), "activity": string(s.recentActivity(8, back))})
			case "error":
				data := map[string]any{"text": e.Text}
				if e.ID != "" {
					data["id"], data["html"], data["status"], data["activity"] = e.ID, s.messageHTML(e.ID, back, true), string(s.statusFor(e.ID)), string(s.recentActivity(8, back))
				}
				send("error", data)
			}
		}
		if over && len(events) == 0 {
			return
		}
		if len(events) > 0 {
			continue
		}
		select {
		case <-changed:
		case <-r.Context().Done():
			return
		}
	}
}

// A liveTurn is a turn under way: what it has done so far, kept so a page
// that comes to it late hears all of it, and the way to stop it.
type liveTurn struct {
	id      string
	started time.Time
	cancel  context.CancelFunc

	mu        sync.Mutex
	events    []chat.Event
	over      bool
	changed   chan struct{}
	listeners int
}

// listen counts a page that follows the turn, coming (1) or going (-1).
func (t *liveTurn) listen(n int) {
	t.mu.Lock()
	t.listeners += n
	t.mu.Unlock()
}

// followed says whether any page is following the turn now.
func (t *liveTurn) followed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.listeners > 0
}

// add records what the turn just did and wakes whoever is following it.
// Events come from the turn and, when the tools run elsewhere, from the
// watch on the log: one at a time.
func (t *liveTurn) add(e chat.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, e)
	close(t.changed)
	t.changed = make(chan struct{})
}

// since is what the turn has done from the i-th event on, whether it is
// over, and what closes when there is more.
func (t *liveTurn) since(i int) ([]chat.Event, bool, <-chan struct{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.events[i:], t.over, t.changed
}

// turns are the live turns under way.
type turns struct {
	mu   sync.Mutex
	n    int
	live map[string]*liveTurn
}

// start begins a turn and the context it runs under.
func (ts *turns) start(parent context.Context) (*liveTurn, context.Context) {
	ctx, cancel := context.WithCancel(parent)
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.live == nil {
		ts.live = map[string]*liveTurn{}
	}
	ts.n++
	t := &liveTurn{id: fmt.Sprintf("turn-%d", ts.n), started: time.Now(), cancel: cancel, changed: make(chan struct{})}
	ts.live[t.id] = t
	return t, ctx
}

// end is a turn over: whoever follows it hears the rest and stops.
func (ts *turns) end(t *liveTurn) {
	t.cancel()
	ts.mu.Lock()
	delete(ts.live, t.id)
	ts.mu.Unlock()
	t.mu.Lock()
	t.over = true
	close(t.changed)
	t.changed = make(chan struct{})
	t.mu.Unlock()
}

// find is the turn named, or the latest under way when none is named.
func (ts *turns) find(id string) *liveTurn {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if id != "" {
		return ts.live[id]
	}
	var latest *liveTurn
	for _, t := range ts.live {
		if latest == nil || t.started.After(latest.started) {
			latest = t
		}
	}
	return latest
}

// stop ends the turn named, or every turn under way when none is.
func (ts *turns) stop(id string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	for k, t := range ts.live {
		if id == "" || k == id {
			t.cancel()
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
