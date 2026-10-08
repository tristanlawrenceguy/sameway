package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/soundtrack"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A recording is written down a part at a time: its sound is copied out
// on this computer as chunks (package soundtrack), each chunk is turned
// into plain sound (by the page, or here for a WAV) and written down in
// turn by one worker, so the engine never runs twice at once, and when
// every part is in they are put together at their times. However long the
// recording, nothing large is held and the page says how far it has come.

// partJob is one part of a recording waiting to be written down.
type partJob struct {
	id        string
	wav       string
	index, of int
	start     float64
}

// recordingRoutes are a recording's own addresses: its captions, its sound
// copied out, and writing it down.
func (s *Server) recordingRoutes(m *http.ServeMux) {
	m.HandleFunc("GET /files/{id}/captions.vtt", s.captions)
	m.HandleFunc("GET /files/{id}/sound", s.soundPlan)
	m.HandleFunc("GET /files/{id}/sound/{n}", s.soundChunk)
	m.HandleFunc("POST /files/{id}/transcribe", s.transcribeFile)
	m.HandleFunc("GET /files/{id}/transcript.srt", s.transcriptFile)
	m.HandleFunc("GET /files/{id}/transcript.txt", s.transcriptFile)
	m.HandleFunc("GET /export/workspace.zip", s.exportEverything)
	m.HandleFunc("GET /export/all.ics", s.exportCalendar)
	m.HandleFunc("GET /export/{file}", s.exportFile)
	m.HandleFunc("GET /export/{type}/{file}", s.exportDocument)
}

// enqueue hands a part to the worker, starting it the first time.
func (s *Server) enqueue(j partJob) {
	s.speech.once.Do(func() {
		s.speech.jobs = make(chan partJob, 1024)
		go func() {
			for j := range s.speech.jobs {
				s.writePart(j)
			}
		}()
	})
	s.speech.jobs <- j
}

// partsDir is where a recording's parts are gathered as they are written.
func (s *Server) partsDir(id string) string {
	return filepath.Join(s.app.Workspace.FilesDir(), id+".parts")
}

// soundDir is where a recording's sound is copied out, in chunks.
func (s *Server) soundDir(id string) string {
	return filepath.Join(s.app.Workspace.FilesDir(), id+".sound")
}

// writePart writes one part down, at its place in the recording, and puts
// the recording together when it was the last.
func (s *Server) writePart(j partJob) {
	defer os.Remove(j.wav)
	cues, err := s.speechKit().Transcribe(context.Background(), j.wav)
	rec, gerr := s.app.Store.Get(FileType, j.id)
	if gerr != nil {
		return
	}
	title, _ := rec.Fields["title"].(string)
	if err != nil {
		s.app.Store.Update(FileType, j.id, map[string]any{"status": "failed", "note": "Could not write it down: " + err.Error()})
		records.Record(s.app.Store, "system", records.Change{Action: "failed", Detail: "writing down " + title + ": " + err.Error()})
		os.RemoveAll(s.partsDir(j.id))
		s.Changed()
		return
	}
	for i := range cues {
		cues[i].Start += j.start
		cues[i].End += j.start
	}
	if j.of == 1 {
		cues = s.whoSpoke(rec, j.wav, j.start, cues) // speakers.go
	}
	dir := s.partsDir(j.id)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, strconv.Itoa(j.index)+".vtt"), []byte(speech.VTT(cues)), 0o644)
	done, _ := filepath.Glob(filepath.Join(dir, "*.vtt"))
	if len(done) < j.of {
		s.app.Store.Update(FileType, j.id, map[string]any{"status": "converting", "note": fmt.Sprintf("Being written down on this computer: %d of %d parts done.", len(done), j.of)})
		s.Changed()
		return
	}
	var all []convert.Cue
	sort.Slice(done, func(a, b int) bool { return partNumber(done[a]) < partNumber(done[b]) })
	for _, p := range done {
		data, _ := os.ReadFile(p)
		all = append(all, convert.ParseVTT(string(data))...)
	}
	os.RemoveAll(dir)
	os.RemoveAll(s.soundDir(j.id))
	s.finishWriting(rec, all)
}

func partNumber(path string) int {
	n, _ := strconv.Atoi(strings.TrimSuffix(filepath.Base(path), ".vtt"))
	return n
}

// finishWriting keeps what was heard in a recording: its WebVTT beside it
// and its text, or a note that nothing was.
func (s *Server) finishWriting(rec *store.Record, cues []convert.Cue) {
	title, _ := rec.Fields["title"].(string)
	if len(cues) == 0 {
		s.app.Store.Update(FileType, rec.ID, map[string]any{"status": "ready", "note": "No speech was heard in it."})
		s.Changed()
		return
	}
	cues = s.namedVoices(rec, cues) // me and them, for a call; voices.go
	if path, ok := s.transcriptPath(rec); ok {
		os.WriteFile(path, []byte(speech.VTT(cues)), 0o644)
	}
	s.app.Store.Update(FileType, rec.ID, map[string]any{"status": "ready", "text": convert.Transcript(cues), "note": "Written down on this computer by " + speech.ModelName + ". Edit the text if it misheard."})
	records.Record(s.app.Store, "system", records.Change{Action: "updated", Component: FileType, ID: rec.ID, Detail: title + ", written down", Href: "/t/" + FileType + "/" + rec.ID})
	s.Changed()
}

// writeWAVHere writes down a recording that is already plain sound, all on
// this computer: cut into 16 kHz chunks and queued, with no browser.
func (s *Server) writeWAVHere(rec *store.Record, path string) error {
	os.RemoveAll(s.partsDir(rec.ID))
	parts, err := speech.SplitWAV(path, s.soundDir(rec.ID), soundtrack.Every)
	if err != nil {
		return err
	}
	s.app.Store.Update(FileType, rec.ID, map[string]any{"status": "converting", "note": "Being written down on this computer."})
	for i, p := range parts {
		s.enqueue(partJob{id: rec.ID, wav: p.Path, index: i, of: len(parts), start: p.Start})
	}
	return nil
}

// soundPlan tells the page how a recording's sound comes to be written
// down: here, for a WAV; in chunks copied out here, each a small stream to
// decode; or whole, for what cannot be copied out.
func (s *Server) soundPlan(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(FileType, r.PathValue("id"))
	if err != nil || !isRecording(rec) {
		http.NotFound(w, r)
		return
	}
	path, ok := s.storedPath(rec)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if strings.EqualFold(filepath.Ext(path), ".wav") {
		json.NewEncoder(w).Encode(map[string]any{"host": true})
		return
	}
	chunks, err := soundtrack.Extract(path, s.soundDir(rec.ID))
	if errors.Is(err, soundtrack.ErrUnsupported) || err != nil {
		json.NewEncoder(w).Encode(map[string]any{"whole": true})
		return
	}
	list := make([]map[string]any, len(chunks))
	for i, c := range chunks {
		list[i] = map[string]any{"url": fmt.Sprintf("/files/%s/sound/%d", rec.ID, i), "start": c.Start}
	}
	json.NewEncoder(w).Encode(map[string]any{"chunks": list})
}

// soundChunk serves one chunk of a recording's sound, copied out.
func (s *Server) soundChunk(w http.ResponseWriter, r *http.Request) {
	id, n := r.PathValue("id"), r.PathValue("n")
	if _, err := strconv.Atoi(n); err != nil || strings.ContainsAny(id, `/\.`) {
		http.NotFound(w, r)
		return
	}
	for ext, typ := range map[string]string{"aac": "audio/aac", "ogg": "audio/ogg", "mp3": "audio/mpeg"} {
		path := filepath.Join(s.soundDir(id), n+"."+ext)
		if _, err := os.Stat(path); err == nil {
			w.Header().Set("Content-Type", typ)
			http.ServeFile(w, r, path)
			return
		}
	}
	http.NotFound(w, r)
}
