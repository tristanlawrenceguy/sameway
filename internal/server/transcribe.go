package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A recording is written down on the computer that hosts the workspace,
// whatever computer that is, and never leaves it. Speech-to-text is not
// part of sameway: the owner gets it once, with one press that says what
// it downloads and from where (package speech), and from then on each
// recording is written down as it arrives. The page's script turns any
// recording its browser can play into the plain sound the engine reads;
// with no script, a WAV is read here.

// Speech is how this server writes recordings down. Tests give their own.
type Speech struct {
	Ready      func() bool
	Install    func(ctx context.Context, progress speech.Progress) error
	Transcribe func(ctx context.Context, wav string) ([]convert.Cue, error)
}

// speechState is speech-to-text on this computer: being got, and which
// recordings are being written down.
type speechState struct {
	mu          sync.Mutex
	kit         *Speech
	given       bool // set by UseSpeech rather than downloaded
	getting     bool
	done, total int64
	failed      string
}

// UseSpeech sets how this server writes recordings down.
func (s *Server) UseSpeech(k Speech) {
	s.speech.mu.Lock()
	s.speech.kit, s.speech.given = &k, true
	s.speech.mu.Unlock()
}

// speechKit is the Speech in use: the downloaded engine, unless set.
func (s *Server) speechKit() *Speech {
	s.speech.mu.Lock()
	defer s.speech.mu.Unlock()
	if s.speech.kit == nil {
		dir, _ := speech.Dir()
		s.speech.kit = &Speech{
			Ready:   func() bool { return dir != "" && speech.Ready(dir) },
			Install: func(ctx context.Context, p speech.Progress) error { return speech.Install(ctx, nil, dir, p) },
			Transcribe: func(ctx context.Context, wav string) ([]convert.Cue, error) {
				return speech.Transcribe(ctx, dir, wav)
			},
		}
	}
	return s.speech.kit
}

// speechGet is the owner's press that gets speech-to-text for this computer.
func (s *Server) speechGet(w http.ResponseWriter, r *http.Request) {
	kit := s.speechKit()
	if kit.Ready() {
		s.tell(w, r, outcome{Title: "Speech-to-text is already on this computer"}, "/t/"+FileType)
		return
	}
	if !speech.Supported() && !s.speech.given {
		s.tell(w, r, outcome{Failed: true, Title: "No speech-to-text for this computer", Text: "Its makers publish none for this kind of computer. A transcript can still be written by hand."}, "/t/"+FileType)
		return
	}
	s.speech.mu.Lock()
	already := s.speech.getting
	if !already {
		s.speech.getting, s.speech.done, s.speech.total, s.speech.failed = true, 0, speech.DownloadSize(), ""
	}
	s.speech.mu.Unlock()
	if !already {
		s.record(r, chat.Change{Action: "started", Detail: "getting speech-to-text for this computer (" + sizeWords(speech.DownloadSize()) + ")"})
		go s.getSpeech(kit)
	}
	s.tell(w, r, outcome{Title: "Getting speech-to-text", Text: "It downloads once, about " + sizeWords(speech.DownloadSize()) + ". Recordings are written down as soon as it is here."}, "/t/"+FileType)
}

// getSpeech downloads and unpacks, saying how far it has got, and then
// writes down the recordings that were waiting for it.
func (s *Server) getSpeech(kit *Speech) {
	last := int64(0)
	err := kit.Install(context.Background(), func(done, total int64) {
		s.speech.mu.Lock()
		s.speech.done, s.speech.total = done, total
		s.speech.mu.Unlock()
		if done-last > total/50 {
			last = done
			s.Changed()
		}
	})
	s.speech.mu.Lock()
	s.speech.getting = false
	if err != nil {
		s.speech.failed = err.Error()
	}
	s.speech.mu.Unlock()
	if err != nil {
		chat.Record(s.app.Store, "system", chat.Change{Action: "failed", Detail: "getting speech-to-text: " + err.Error()})
	} else {
		chat.Record(s.app.Store, "system", chat.Change{Action: "added", Detail: "speech-to-text for this computer (" + speech.ModelName + ")"})
	}
	s.Changed()
}

// transcribeFile writes one recording down. The page's script sends the
// recording as 16 kHz WAV; with no script the original is used, when it
// is a WAV.
func (s *Server) transcribeFile(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(FileType, r.PathValue("id"))
	if err != nil || rec.Fields["kind"] != "audio" {
		http.NotFound(w, r)
		return
	}
	back := "/t/" + FileType + "/" + rec.ID
	if !s.speechKit().Ready() {
		s.tell(w, r, outcome{Failed: true, Title: "Not written down", Text: "Speech-to-text is not on this computer yet."}, back)
		return
	}
	var sound io.Reader
	if strings.HasPrefix(r.Header.Get("Content-Type"), "audio/wav") {
		sound = http.MaxBytesReader(w, r.Body, 1<<30)
	} else if path, ok := s.storedPath(rec); ok && strings.EqualFold(filepath.Ext(path), ".wav") {
		f, err := os.Open(path)
		if err == nil {
			defer f.Close()
			sound = f
		}
	}
	if sound == nil {
		s.tell(w, r, outcome{Failed: true, Title: "Not written down", Text: "A " + strings.ToUpper(convert.Ext(fmt.Sprint(rec.Fields["name"]))) + " recording is read by this page's script, which is off. Turn scripts on, or add it as a WAV."}, back)
		return
	}
	samples, rate, err := speech.ReadWAV(sound)
	if err != nil {
		s.tell(w, r, outcome{Failed: true, Title: "Not written down", Text: err.Error()}, back)
		return
	}
	var buf bytes.Buffer
	speech.WriteWAV(&buf, speech.Resample(samples, rate))
	wav := filepath.Join(s.app.Workspace.FilesDir(), rec.ID+".speech.wav")
	if err := os.WriteFile(wav, buf.Bytes(), 0o644); err != nil {
		s.failed(w, r, "Not written down", err, back)
		return
	}
	s.app.Store.Update(FileType, rec.ID, map[string]any{"status": "converting", "note": "Being written down on this computer."})
	go s.writeDown(rec, wav)
	s.tellAt(w, r, outcome{Title: "Writing it down", Text: "The transcript appears here when it is done."}, back)
}

// writeDown runs the engine on one recording and keeps what it heard.
func (s *Server) writeDown(rec *store.Record, wav string) {
	defer os.Remove(wav)
	cues, err := s.speechKit().Transcribe(context.Background(), wav)
	title, _ := rec.Fields["title"].(string)
	switch {
	case err != nil:
		s.app.Store.Update(FileType, rec.ID, map[string]any{"status": "failed", "note": "Could not write it down: " + err.Error()})
		chat.Record(s.app.Store, "system", chat.Change{Action: "failed", Detail: "writing down " + title + ": " + err.Error()})
	case len(cues) == 0:
		s.app.Store.Update(FileType, rec.ID, map[string]any{"status": "ready", "note": "No speech was heard in it."})
	default:
		if path, ok := s.transcriptPath(rec); ok {
			os.WriteFile(path, []byte(speech.VTT(cues)), 0o644)
		}
		s.app.Store.Update(FileType, rec.ID, map[string]any{"status": "ready", "text": convert.Transcript(cues), "note": "Written down on this computer by " + speech.ModelName + ". Edit the text if it misheard."})
		chat.Record(s.app.Store, "system", chat.Change{Action: "updated", Component: FileType, ID: rec.ID, Detail: title + ", written down", Href: "/t/" + FileType + "/" + rec.ID})
	}
	s.Changed()
}

// speechOffer is what a recording's page offers about writing it down: to
// get speech-to-text (its owner), to write it down now, or how far the
// getting has come.
func (s *Server) speechOffer(r *http.Request, rec *store.Record, props map[string]any) string {
	if _, has := props["cues"]; has || rec.Fields["status"] == "converting" {
		return ""
	}
	kit := s.speechKit()
	s.speech.mu.Lock()
	getting, done, total, failed := s.speech.getting, s.speech.done, s.speech.total, s.speech.failed
	s.speech.mu.Unlock()
	switch {
	case kit.Ready():
		props["make"] = map[string]any{"action": "/files/" + rec.ID + "/transcribe", "auto": rec.Fields["status"] != "failed"}
	case getting:
		return string(s.component("status", map[string]any{"id": "speech-status", "state": "working",
			"message": fmt.Sprintf("Getting speech-to-text for this computer: %s of %s.", sizeWords(done), sizeWords(total))}))
	case chat.VisitorOf(r.Context()).Owner() && (speech.Supported() || s.speech.given):
		if failed != "" {
			props["none"] = "No transcript yet. Getting speech-to-text did not work: " + failed
		}
		props["get"] = map[string]any{"action": "/speech/get", "size": sizeWords(speech.DownloadSize())}
	default:
		props["none"] = "No transcript yet. Until there is one, someone who cannot hear it gets only its title. The owner of this workspace can get speech-to-text for this computer."
	}
	return ""
}
