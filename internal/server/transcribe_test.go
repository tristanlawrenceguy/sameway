package server_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
)

// A recording is written down on this computer: the owner is offered
// speech-to-text, saying what it downloads, and nobody else is; once it
// is here the page writes each recording down by itself, from the sound
// its script sends, and the transcript becomes the recording's words on
// its page and the file's text.
func TestARecordingIsWrittenDownOnThisComputer(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)
	var ready atomic.Bool
	heard := make(chan string, 1)
	srv.UseSpeech(server.Speech{
		Ready: func() bool { return ready.Load() },
		Install: func(ctx context.Context, p speech.Progress) error {
			p(50, 100)
			ready.Store(true)
			return nil
		},
		Transcribe: func(ctx context.Context, wav string) ([]convert.Cue, error) {
			data, _ := os.ReadFile(wav)
			samples, rate, err := speech.ReadWAV(bytes.NewReader(data))
			if err != nil || rate != speech.Rate {
				heard <- "not 16 kHz"
				return nil, err
			}
			heard <- "ok"
			_ = samples
			return []convert.Cue{{Start: 0.3, End: 2.1, Text: "Order the compost."}, {Start: 2.5, End: 4, Text: "Dig the pond."}}, nil
		},
	})
	body, ct := multipartFile(t, "Memo.m4a", "not really audio", nil)
	res := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
	loc := res.Header().Get("Location")
	id := strings.TrimPrefix(loc, "/t/file/")

	page := get(t, h, loc).Body.String()
	if !strings.Contains(page, `action="/speech/get"`) || !strings.Contains(page, "Recordings never leave this computer") || !strings.Contains(page, "downloaded once from GitHub and Hugging Face") {
		t.Errorf("the owner is offered speech-to-text, saying what it fetches:\n%.3000s", page)
	}
	viewer := chat.Visitor{Name: "Hana", Login: "hana@example.com", Access: chat.Edit}
	if theirs := as(t, h, viewer, http.MethodGet, loc, "", "").Body.String(); strings.Contains(theirs, `action="/speech/get"`) || !strings.Contains(theirs, "The owner of this workspace can get speech-to-text") {
		t.Error("only the owner is offered it; others are told who can")
	}
	if got := as(t, h, viewer, http.MethodPost, "/speech/get", "", ""); got.Code == http.StatusSeeOther && ready.Load() {
		t.Error("only the owner may get it")
	}

	wantStatus(t, do(t, h, http.MethodPost, "/speech/get", nil, ""), http.StatusSeeOther)
	for i := 0; i < 100 && !ready.Load(); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	page = get(t, h, loc).Body.String()
	if !strings.Contains(page, `<form class="sw-audio__make" method="post" action="/files/`+id+`/transcribe" data-auto>`) {
		t.Errorf("once it is here, the page writes the recording down by itself:\n%.3000s", page)
	}

	// The page's script sends the sound as 16 kHz WAV; any WAV is brought to it.
	var wav bytes.Buffer
	speech.WriteWAV(&wav, make([]float32, 8000))
	req := httptest.NewRequest(http.MethodPost, "/files/"+id+"/transcribe", &wav)
	req.Header.Set("Content-Type", "audio/wav")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	wantStatus(t, rec, http.StatusSeeOther)
	select {
	case got := <-heard:
		if got != "ok" {
			t.Fatal(got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the engine was never run")
	}
	var file map[string]any
	for i := 0; i < 200; i++ {
		f, _ := a.Store.Get("file", id)
		file = f.Fields
		if file["status"] == "ready" && file["text"] != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(file["text"].(string), "[0:00] Order the compost.\n\n[0:02] Dig the pond.") {
		t.Errorf("the transcript is the file's text: %v", file)
	}
	stored := file["path"].(string)
	if _, err := os.Stat(filepath.Join(a.Workspace.FilesDir(), strings.TrimSuffix(stored, filepath.Ext(stored))+".vtt")); err != nil {
		t.Error("and kept beside the original as WebVTT")
	}
	gone := false
	for i := 0; i < 200 && !gone; i++ {
		_, err := os.Stat(filepath.Join(a.Workspace.FilesDir(), id+".speech.wav"))
		gone = err != nil
		time.Sleep(10 * time.Millisecond)
	}
	if !gone {
		t.Error("the sound sent to be written down is not kept")
	}
	page = get(t, h, loc).Body.String()
	if !strings.Contains(page, `data-start="2.00"`) || !strings.Contains(page, "Dig the pond.") || strings.Contains(page, "sw-audio__make") {
		t.Error("the page shows the transcript, and offers nothing more to write down")
	}

	// With no script, only a WAV can be sent; anything else says why.
	body, ct = multipartFile(t, "Other.m4a", "x", nil)
	other := strings.TrimPrefix(do(t, h, http.MethodPost, "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
	back := after(t, h, do(t, h, http.MethodPost, "/files/"+other+"/transcribe", nil, "")).Body.String()
	if !strings.Contains(back, "read by this page") {
		t.Errorf("an M4A with scripts off says why it was not written down: %.2000s", back)
	}
}
