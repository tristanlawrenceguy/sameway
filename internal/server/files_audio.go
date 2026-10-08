package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A recording on its file's page, heard or seen: played by the browser's
// own player, or the media component's bar where its script runs, with its
// transcript under it on the same page, each line leading to where it was
// said, and on a video as its captions. The transcript is kept beside the
// original as <id>.vtt, whoever made it.

// isRecording says a file is something heard: a recording or a video.
func isRecording(rec *store.Record) bool {
	kind, _ := rec.Fields["kind"].(string)
	return kind == "audio" || kind == "video"
}

// headOf is the first 64 KB of a kept file, enough to tell what is in it.
func (s *Server) headOf(id string) []byte {
	rec, err := s.app.Store.Get(FileType, id)
	if err != nil {
		return nil
	}
	path, ok := s.storedPath(rec)
	if !ok {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	head := make([]byte, 64<<10)
	n, _ := io.ReadFull(f, head)
	return head[:n]
}

// transcriptPath is where a recording's transcript is kept, when it has one.
func (s *Server) transcriptPath(rec *store.Record) (string, bool) {
	path, ok := s.storedPath(rec)
	if !ok {
		return "", false
	}
	return strings.TrimSuffix(path, filepath.Ext(path)) + ".vtt", true
}

// recordingOf is the media component's props for a recording or a video.
func (s *Server) recordingOf(rec *store.Record) map[string]any {
	title, _ := rec.Fields["title"].(string)
	name, _ := rec.Fields["name"].(string)
	src := "/files/" + rec.ID
	kind, _ := rec.Fields["kind"].(string)
	props := map[string]any{"id": "media-" + rec.ID, "kind": kind, "title": title, "src": src, "type": convert.MediaType(name, kind)}
	about := strings.ToUpper(convert.Ext(name))
	if n, ok := rec.Fields["size"].(int64); ok && n > 0 {
		about += " · " + sizeWords(n)
	} else if n, ok := rec.Fields["size"].(int); ok && n > 0 {
		about += " · " + sizeWords(int64(n))
	}
	props["about"] = about
	if cues := s.heard(rec); len(cues) > 0 {
		list := make([]any, 0, len(cues))
		anchored := map[int]bool{}
		for _, c := range cues {
			cue := map[string]any{"start": fmt.Sprintf("%.2f", c.Start), "at": convert.Clock(c.Start), "said": convert.Spoken(c.Start), "text": c.Text}
			// A line is linked to by its second, as a meeting's write-up
			// does (chat/meeting.go); the first line in a second has it.
			if sec := int(c.Start); !anchored[sec] {
				anchored[sec], cue["anchor"] = true, fmt.Sprint(sec)
			}
			if c.Speaker != "" {
				cue["speaker"] = c.Speaker
			}
			list = append(list, cue)
		}
		props["cues"] = list
		if kind == "video" {
			props["captions"] = src + "/captions.vtt"
		}
		props["downloads"] = []any{
			map[string]any{"href": src + "/transcript.srt", "format": "srt"},
			map[string]any{"href": src + "/transcript.txt", "format": "txt"},
		}
	}
	return props
}

// heard is a recording's transcript: its text, which people and the
// assistant correct, when it reads as one, else the WebVTT beside it as
// it was first written down.
func (s *Server) heard(rec *store.Record) []convert.Cue {
	if !isRecording(rec) {
		return nil
	}
	if text, _ := rec.Fields["text"].(string); text != "" {
		if cues := convert.FromTranscript(text); len(cues) > 0 {
			return cues
		}
	}
	if path, ok := s.transcriptPath(rec); ok {
		if data, err := os.ReadFile(path); err == nil {
			return convert.ParseVTT(string(data))
		}
	}
	return nil
}

// captions is a recording's transcript as WebVTT, made from its text so a
// correction shows on the picture too: each line until the next begins,
// the last for a few seconds.
func (s *Server) captions(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(FileType, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cues := s.heard(rec)
	if len(cues) == 0 {
		http.NotFound(w, r)
		return
	}
	for i := range cues {
		if cues[i].End <= cues[i].Start {
			cues[i].End = cues[i].Start + 5
			if i+1 < len(cues) && cues[i+1].Start > cues[i].Start {
				cues[i].End = cues[i+1].Start
			}
		}
	}
	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	io.WriteString(w, speech.VTT(cues))
}

// pairCaptions gives subtitles to the recording they belong to: one of the
// same name with no words yet, as its transcript and, for a video, its
// captions. Subtitles for none stay a file of their own.
func (s *Server) pairCaptions(id string, data []byte) {
	sub, err := s.app.Store.Get(FileType, id)
	if err != nil {
		return
	}
	title, _ := sub.Fields["title"].(string)
	cues := convert.ParseVTT(string(data))
	if len(cues) == 0 || title == "" {
		return
	}
	recs, _ := s.app.Store.List(FileType, store.ListOptions{})
	for _, rec := range recs {
		other, _ := rec.Fields["title"].(string)
		if rec.ID == id || !isRecording(rec) || !strings.EqualFold(other, title) || len(s.heard(rec)) > 0 {
			continue
		}
		if path, ok := s.transcriptPath(rec); ok {
			os.WriteFile(path, []byte(speech.VTT(cues)), 0o644)
		}
		name, _ := sub.Fields["name"].(string)
		s.app.Store.Update(FileType, rec.ID, map[string]any{"text": convert.Transcript(cues), "note": "Its words are from " + name + ". Edit the text if they are wrong."})
		s.app.Store.Update(FileType, id, map[string]any{"note": "These are the words of " + other + ", given to it."})
		s.Changed()
		return
	}
}
