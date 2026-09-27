package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A recording on its file's page: played by the browser's own player,
// or the audio component's bar where its script runs, with its transcript
// under it on the same page, each line leading to where it was said.
// The transcript is kept beside the original as <id>.vtt, whoever made it.

// transcriptPath is where a recording's transcript is kept, when it has one.
func (s *Server) transcriptPath(rec *store.Record) (string, bool) {
	path, ok := s.storedPath(rec)
	if !ok {
		return "", false
	}
	return strings.TrimSuffix(path, filepath.Ext(path)) + ".vtt", true
}

// recordingOf is the audio component's props for a file that is a recording.
func (s *Server) recordingOf(rec *store.Record) map[string]any {
	title, _ := rec.Fields["title"].(string)
	name, _ := rec.Fields["name"].(string)
	src := "/files/" + rec.ID
	props := map[string]any{"id": "audio-" + rec.ID, "title": title, "src": src, "type": convert.AudioType(name)}
	about := strings.ToUpper(convert.Ext(name))
	if n, ok := rec.Fields["size"].(int64); ok && n > 0 {
		about += " · " + sizeWords(n)
	} else if n, ok := rec.Fields["size"].(int); ok && n > 0 {
		about += " · " + sizeWords(int64(n))
	}
	props["about"] = about
	if cues := s.heard(rec); len(cues) > 0 {
		list := make([]any, 0, len(cues))
		for _, c := range cues {
			cue := map[string]any{"start": fmt.Sprintf("%.2f", c.Start), "at": convert.Clock(c.Start), "said": convert.Spoken(c.Start), "text": c.Text}
			if c.Speaker != "" {
				cue["speaker"] = c.Speaker
			}
			list = append(list, cue)
		}
		props["cues"] = list
	}
	return props
}

// heard is a recording's transcript: its text, which people and the
// assistant correct, when it reads as one, else the WebVTT beside it as
// it was first written down.
func (s *Server) heard(rec *store.Record) []convert.Cue {
	if kind, _ := rec.Fields["kind"].(string); kind != "audio" {
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

// sizeWords is a file's size as a person reads it: 11 MB, 480 KB.
func sizeWords(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.0f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}
