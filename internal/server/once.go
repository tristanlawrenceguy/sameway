package server

import (
	"bytes"
	"crypto/sha256"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A browser does not send a form twice; an agent whose request timed out
// sends it again, and without this the note it asked for is made twice
// with nothing to say the first one worked. A change sent with an
// Idempotency-Key header is done once: the same key again, from the same
// caller, gets the first answer back; a repeat that arrives while the
// first is still being done waits for it; and the same key for a
// different request is refused, since that is a mistake. Answers are kept
// a day, while this server runs.

const (
	onceKept  = 24 * time.Hour
	onceMax   = 10000
	onceBytes = 1 << 20
)

type onceAnswer struct {
	sum    [32]byte // the request, to tell a repeat from a reuse
	status int
	header http.Header
	body   []byte
	at     time.Time
	done   chan struct{}
}

type onceKeys struct {
	mu   sync.Mutex
	seen map[string]*onceAnswer
}

var once = &onceKeys{seen: map[string]*onceAnswer{}}

// doOnce serves r through next, or answers it as its first sending was
// answered.
func (s *Server) doOnce(w http.ResponseWriter, r *http.Request, next func(http.ResponseWriter, *http.Request)) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || r.Method == http.MethodGet || r.Method == http.MethodHead {
		next(w, r)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	r.Body = io.NopCloser(bytes.NewReader(body))
	v := records.VisitorOf(r.Context())
	sum := sha256.Sum256(append([]byte(r.Method+" "+r.URL.RequestURI()+"\n"), body...))
	id := s.app.Workspace.Dir + "|" + v.Login + "|" + string(v.Access) + "|" + key

	once.mu.Lock()
	for k, a := range once.seen {
		if time.Since(a.at) > onceKept || len(once.seen) > onceMax {
			delete(once.seen, k)
		}
	}
	first, seen := once.seen[id]
	if !seen {
		first = &onceAnswer{sum: sum, at: time.Now(), done: make(chan struct{})}
		once.seen[id] = first
	}
	once.mu.Unlock()

	if seen {
		if first.sum != sum {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": apiError{Code: "key_reused",
				Message: "that Idempotency-Key was used for a different request; use a new key for each new change, and the same one only to send the same change again"}})
			return
		}
		<-first.done
		once.mu.Lock()
		kept := once.seen[id] == first
		once.mu.Unlock()
		if !kept {
			next(w, r) // the first answer was not kept, so this is done again
			return
		}
		for k, vals := range first.header {
			w.Header()[k] = vals
		}
		w.Header().Set("Idempotent-Replayed", "true")
		w.WriteHeader(first.status)
		w.Write(first.body)
		return
	}
	rec := &onceWriter{w: w, status: http.StatusOK}
	defer func() {
		failed := recover()
		first.status, first.header, first.body = rec.status, w.Header().Clone(), rec.buf.Bytes()
		if rec.big || strings.HasPrefix(w.Header().Get("Content-Type"), "text/event-stream") || failed != nil {
			// A stream, a large answer or one that failed on the way is
			// not kept: a repeat is done again.
			once.mu.Lock()
			delete(once.seen, id)
			once.mu.Unlock()
		}
		close(first.done)
		if failed != nil {
			panic(failed)
		}
	}()
	next(rec, r)
}

// onceWriter passes an answer on and keeps a copy of it.
type onceWriter struct {
	w      http.ResponseWriter
	status int
	buf    bytes.Buffer
	big    bool
}

func (o *onceWriter) Header() http.Header { return o.w.Header() }
func (o *onceWriter) WriteHeader(code int) {
	o.status = code
	o.w.WriteHeader(code)
}
func (o *onceWriter) Write(p []byte) (int, error) {
	if o.buf.Len()+len(p) > onceBytes {
		o.big = true
	} else {
		o.buf.Write(p)
	}
	return o.w.Write(p)
}
func (o *onceWriter) Flush() {
	if f, ok := o.w.(http.Flusher); ok {
		f.Flush()
	}
}
