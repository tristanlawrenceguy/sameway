package server

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/speech"
)

// Saying a message instead of typing it: the page records, turns the
// recording into plain sound, and sends it here; it is written down on
// this computer, as recordings are, and the words go back into the
// message to be read before they are sent. Nothing is kept.

// voiceFor is the chat's microphone: Record for a voice note to attach,
// always, and Dictate and voice mode once speech-to-text is on this
// computer.
func (s *Server) voiceFor() (record, dictate, talk template.HTML) {
	record = s.component("voice", map[string]any{"mode": "record", "target": "attach", "context": "a voice note to attach"})
	if s.speechKit().Ready() {
		dictate = s.component("voice", map[string]any{"mode": "dictate", "target": "message", "action": "/dictate"})
		talk = s.component("talk", map[string]any{"message": "message", "action": "/dictate"})
	}
	return record, dictate, talk
}

// dictate writes down a short recording sent as WAV and answers its words.
func (s *Server) dictate(w http.ResponseWriter, r *http.Request) {
	answer := func(code int, body map[string]string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(body)
	}
	if !s.speechKit().Ready() {
		answer(http.StatusConflict, map[string]string{"error": "Speech-to-text is not on this computer yet. Its owner can get it from any recording's page."})
		return
	}
	samples, rate, err := speech.ReadWAV(http.MaxBytesReader(w, r.Body, 64<<20))
	if err != nil {
		log.Printf("dictate: %v", err)
		answer(http.StatusBadRequest, map[string]string{"error": "The sound could not be read, so it was not written down."})
		return
	}
	var buf bytes.Buffer
	speech.WriteWAV(&buf, speech.Resample(samples, rate))
	f, err := os.CreateTemp("", "sameway-dictate-*.wav")
	if err != nil {
		log.Printf("dictate: %v", err)
		answer(http.StatusInternalServerError, map[string]string{"error": "It could not be written down on this computer."})
		return
	}
	path := f.Name()
	defer os.Remove(path)
	f.Write(buf.Bytes())
	f.Close()
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	cues, err := s.speechKit().Transcribe(ctx, filepath.Clean(path))
	if err != nil {
		log.Printf("dictate: %v", err)
		answer(http.StatusInternalServerError, map[string]string{"error": "It could not be written down on this computer."})
		return
	}
	var words []string
	for _, c := range cues {
		words = append(words, c.Text)
	}
	answer(http.StatusOK, map[string]string{"text": strings.Join(words, " ")})
}
