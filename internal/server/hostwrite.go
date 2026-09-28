package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/look"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Recordings are written down on the computer that hosts the workspace
// with no page open: a WAV straight away, anything else in the Chrome or
// Edge on that computer, run in the background to read the sound the way
// a page would. Whatever way a recording arrives (a page, the command
// line, the assistant) it is written down once speech-to-text is here,
// and the page only does it itself where the host cannot.

// hostState is writing down in the background: whether it is on, which
// recordings are waiting or being done, and which the host could not do.
type hostState struct {
	mu      sync.Mutex
	on      bool
	waiting map[string]bool
	failed  map[string]bool
	queue   chan string
}

// WriteDownInBackground has this server write recordings down itself, a
// sweep a minute for any that arrived without it (the command line, a
// copy synced from another computer). A running workspace turns it on;
// tests do not, so nothing starts a browser behind them.
func (s *Server) WriteDownInBackground() {
	s.host.mu.Lock()
	if s.host.on {
		s.host.mu.Unlock()
		return
	}
	s.host.on = true
	s.host.waiting, s.host.failed = map[string]bool{}, map[string]bool{}
	s.host.queue = make(chan string, 4096)
	s.host.mu.Unlock()
	go func() {
		for id := range s.host.queue {
			s.hostWrite(id)
		}
	}()
	go func() {
		for {
			s.sweep()
			time.Sleep(time.Minute)
		}
	}()
}

// hostWrites says whether this server will write a recording down itself,
// so its page need not.
func (s *Server) hostWrites(rec *store.Record) bool {
	s.host.mu.Lock()
	defer s.host.mu.Unlock()
	return s.host.on && !s.host.failed[rec.ID]
}

// hostWaiting says a recording is waiting to be written down here.
func (s *Server) hostWaiting(id string) bool {
	s.host.mu.Lock()
	defer s.host.mu.Unlock()
	return s.host.waiting[id]
}

// writeLater queues a recording to be written down here, once.
func (s *Server) writeLater(id string) {
	s.host.mu.Lock()
	defer s.host.mu.Unlock()
	if !s.host.on || s.host.waiting[id] || s.host.failed[id] {
		return
	}
	s.host.waiting[id] = true
	s.host.queue <- id
}

// sweep queues every recording that has no words yet and is not being
// written down, once speech-to-text is here.
func (s *Server) sweep() {
	if !s.speechKit().Ready() {
		return
	}
	recs, err := s.app.Store.List(FileType, store.ListOptions{})
	if err != nil {
		return
	}
	for _, rec := range recs {
		note, _ := rec.Fields["note"].(string)
		if isRecording(rec) && rec.Fields["status"] == "ready" && len(s.heard(rec)) == 0 && !strings.HasPrefix(note, "No speech was heard") {
			s.writeLater(rec.ID)
		}
	}
}

// hostWrite writes one recording down here: a WAV by cutting it up, and
// anything else by reading its sound in the background browser, sending
// each part as a page would. When the host cannot, the page is let do it.
func (s *Server) hostWrite(id string) {
	defer func() {
		s.host.mu.Lock()
		delete(s.host.waiting, id)
		s.host.mu.Unlock()
	}()
	rec, err := s.app.Store.Get(FileType, id)
	if err != nil || !isRecording(rec) || len(s.heard(rec)) > 0 {
		return
	}
	path, ok := s.storedPath(rec)
	if !ok {
		return
	}
	if strings.EqualFold(filepath.Ext(path), ".wav") {
		err = s.writeWAVHere(rec, path)
	} else {
		err = s.readInBackground(id)
	}
	if err != nil {
		log.Printf("writing down %s here: %v; its page will do it", id, err)
		s.host.mu.Lock()
		s.host.failed[id] = true
		s.host.mu.Unlock()
		s.Changed()
	}
}

// readInBackground has the background browser read a recording's sound a
// chunk at a time and send each part to be written down, on a port of
// this computer's own for as long as it takes.
func (s *Server) readInBackground(id string) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: s.mux}
	go srv.Serve(ln)
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()
	s.app.Store.Update(FileType, id, map[string]any{"status": "converting", "note": "Being written down on this computer."})
	s.Changed()
	out, err := look.RunScript(ctx, "http://"+ln.Addr().String()+"/search", fmt.Sprintf(readScript, id))
	if err != nil {
		s.app.Store.Update(FileType, id, map[string]any{"status": "ready", "note": "A recording's text is its transcript; there is none yet."})
		if errors.Is(err, look.ErrNoBrowser) {
			return err
		}
		return fmt.Errorf("the background browser could not read it: %w", err)
	}
	log.Printf("writing down %s here: %s", id, out)
	return nil
}

// readScript is what the background browser runs: the page's own way of
// reading a recording (swSpeech, 24-speech.js), chunk by chunk.
const readScript = `(async () => {
  const id = %q, at = '/files/' + id;
  const plan = await (await fetch(at + '/sound')).json();
  const chunks = plan.whole ? [{ url: at, start: 0 }] : plan.chunks;
  for (let i = 0; i < chunks.length; i++) {
    const wav = await window.swSpeech.toWav(chunks[i].url);
    const q = chunks.length > 1 ? '?part=' + i + '&of=' + chunks.length + '&start=' + chunks[i].start : '';
    const r = await fetch(at + '/transcribe' + q, { method: 'POST', body: wav, headers: { 'Content-Type': 'audio/wav' } });
    if (!r.ok) throw new Error('the host refused part ' + (i + 1) + ': ' + r.status);
  }
  return 'sent ' + chunks.length + ' part(s)';
})()`
