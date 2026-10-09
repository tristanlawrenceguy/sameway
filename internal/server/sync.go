package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/peers"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Other computers that host this workspace keep in step with it here; see
// internal/peers. Only the owner's own computers and the people the owner
// made hosts may: a copy can change anything, access included.

func (s *Server) syncExchange(w http.ResponseWriter, r *http.Request) {
	if v := records.VisitorOf(r.Context()); !v.Owner() && v.Access != records.Host {
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
	s.HearPresence(in.Present)
	out.Present = s.PresentHere()
	if n > 0 {
		s.Changed()
	}
	writeJSON(w, http.StatusOK, out)
}

// Changed tells every open page that the workspace changed from elsewhere,
// so it follows: another computer's change, arrived by sync.
func (s *Server) Changed() { s.changes.Add(1) }

// fresh takes the content types as schema/ has them now, before a page or
// an answer is made from them, so a type or field another process made
// (the assistant's tools under claude-code run in `sameway mcp`) is there
// at once, not after a restart. It costs a look at the folder; open pages
// hear of a change from WatchSchema, or from here, whichever sees it first.
func (s *Server) fresh() {
	ok, err := s.app.ReloadSchema()
	if err != nil {
		log.Printf("schema: %v", err)
	}
	if ok {
		s.Changed()
	}
}

// events is an open page's one connection (01-connect.js). It tells the
// page, as a server-sent event, each time the workspace changes from
// elsewhere: the page fetches itself and moves what changed into place
// (17-refresh.js), as it does after a turn. With ?ring=1, a page with a
// clock, it also tells each reminder as it rings (clock.go), checked every
// few seconds.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	if isPublic(r) {
		http.NotFound(w, r) // a published page follows by being read again
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
	fmt.Fprint(w, "event: hello\ndata: {}\n\n")
	flusher.Flush()
	last, others := s.changes.Load(), s.presentFor(r)
	idle, rings := r.URL.Query().Get("idle") != "", r.URL.Query().Get("ring") != ""
	told := map[string]bool{}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for n := 0; ; n++ {
		if rings && n%5 == 0 {
			s.tellRings(w, told)
			flusher.Flush()
		}
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
		}
		// An open page is someone here, unless it says its person has
		// been idle a while (01-connect.js); who else is here changing is
		// news to the page, like a change.
		if !idle {
			s.seen(r, refererPath(r))
		}
		now, who := s.changes.Load(), s.presentFor(r)
		if now != last || who != others {
			last, others = now, who
			fmt.Fprint(w, "event: changed\ndata: {}\n\n")
			flusher.Flush()
		}
	}
}
