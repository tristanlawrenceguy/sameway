package server

import (
	"net/http"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// whenRead says how words are read as a day or a moment, the one reading
// the server keeps: {text, day} when they are read, {error} with what they
// must be when not. A when-field says it back as it is typed. With repeat
// instead of words it reads how often: {text, rule}, "every Tuesday" and
// what is kept, or {error}; never is no repeat, "once, not again".
func (s *Server) whenRead(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	if r.URL.Query().Has("repeat") {
		words := r.URL.Query().Get("repeat")
		rule, ok := when.ParseRepeat(words, now)
		if !ok {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": when.RepeatWhy(words, now)})
			return
		}
		text := when.RepeatText(rule)
		if rule == "" {
			text = "once, not again"
		}
		writeJSON(w, http.StatusOK, map[string]any{"text": text, "rule": rule})
		return
	}
	words := r.URL.Query().Get("words")
	t, day, ok := when.Parse(words, now)
	if !ok {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": when.Why(words, now)})
		return
	}
	stored := when.Store(t, day)
	writeJSON(w, http.StatusOK, map[string]any{"text": when.Text(stored), "day": stored[:10]})
}
