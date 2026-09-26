package server

import (
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// whenRead says how words are read as a day or a moment, the one reading
// the server keeps: {text, day} when they are read, {error} with what they
// must be when not. A when-field says it back as it is typed.
func (s *Server) whenRead(w http.ResponseWriter, r *http.Request) {
	words := r.URL.Query().Get("words")
	now := time.Now()
	t, day, ok := when.Parse(words, now)
	if !ok {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": when.Why(words, now)})
		return
	}
	stored := when.Store(t, day)
	writeJSON(w, http.StatusOK, map[string]any{"text": when.Text(stored), "day": stored[:10]})
}
