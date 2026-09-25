package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/peers"
)

// Other computers that host this workspace keep in step with it here; see
// internal/peers. Only the owner's own computers and the people the owner
// made hosts may: a copy can change anything, access included.

func (s *Server) syncExchange(w http.ResponseWriter, r *http.Request) {
	if v := chat.VisitorOf(r.Context()); !v.Owner() && v.Access != chat.Host {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "only the computers that host this workspace keep in step with it"})
		return
	}
	var in peers.Message
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "send JSON with seen and stamps"})
		return
	}
	out, n, err := peers.Answer(s.app.Store, in)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if n > 0 {
		s.Changed()
	}
	writeJSON(w, http.StatusOK, out)
}

// Changed tells every open page that the workspace changed from elsewhere,
// so it follows: another computer's change, arrived by sync.
func (s *Server) Changed() { s.changes.Add(1) }

// events tells an open page, as a server-sent event, each time the
// workspace changes from elsewhere. The page fetches itself and moves what
// changed into place (17-refresh.js), as it does after a turn.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not possible here", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "event: hello\ndata: {}\n\n")
	flusher.Flush()
	last := s.changes.Load()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
		}
		if now := s.changes.Load(); now != last {
			last = now
			fmt.Fprint(w, "event: changed\ndata: {}\n\n")
			flusher.Flush()
		}
	}
}
