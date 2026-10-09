package media_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
)

// The chat offers a voice note to attach always, and dictation once
// speech-to-text is on this computer; dictation writes the sound down here
// and answers its words, keeping nothing; without speech-to-text it says
// who can get it.
func TestAMessageCanBeSaidInsteadOfTyped(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	srv := h.(*server.Server)
	var ready atomic.Bool
	srv.UseSpeech(server.Speech{
		Ready: func() bool { return ready.Load() },
		Transcribe: func(ctx context.Context, wav string) ([]convert.Cue, error) {
			return []convert.Cue{{Start: 0, End: 1, Text: "Add a task to"}, {Start: 1, End: 2, Text: "order compost."}}, nil
		},
	})

	page := get(t, h, "/chat").Body.String()
	if !strings.Contains(page, `data-component="voice" data-mode="record" data-target="attach"`) {
		t.Error("the chat offers a voice note to attach")
	}
	if strings.Contains(page, `data-mode="dictate"`) || strings.Contains(page, `data-component="talk"`) {
		t.Error("dictation is offered only once speech-to-text is here")
	}
	var wav bytes.Buffer
	speech.WriteWAV(&wav, make([]float32, 16000))
	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/dictate", bytes.NewReader(wav.Bytes()))
		req.Header.Set("Content-Type", "audio/wav")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if res := send(); res.Code != http.StatusConflict || !strings.Contains(res.Body.String(), "Its owner can get it") {
		t.Errorf("without speech-to-text dictation says who can get it: %d %s", res.Code, res.Body.String())
	}

	ready.Store(true)
	page = get(t, h, "/chat").Body.String()
	if !strings.Contains(page, `data-mode="dictate" data-target="message" data-action="/dictate"`) {
		t.Error("with speech-to-text here, the message can be dictated")
	}
	if !strings.Contains(page, `<section class="sw-talk" data-component="talk" data-message="message" data-action="/dictate" data-state="off"`) || !strings.Contains(page, "Start voice mode") {
		t.Error("and voice mode is offered, hidden until the page can listen")
	}
	res := send()
	wantStatus(t, res, http.StatusOK)
	var got struct{ Text string }
	json.Unmarshal(res.Body.Bytes(), &got)
	if got.Text != "Add a task to order compost." {
		t.Errorf("the words come back as one message, not %q", got.Text)
	}
	req := httptest.NewRequest(http.MethodPost, "/dictate", strings.NewReader("not a wav"))
	bad := httptest.NewRecorder()
	h.ServeHTTP(bad, req)
	if bad.Code != http.StatusBadRequest {
		t.Errorf("sound that cannot be read is refused, got %d", bad.Code)
	}
	var refusal struct{ Error string }
	json.Unmarshal(bad.Body.Bytes(), &refusal)
	if refusal.Error != "The sound could not be read, so it was not written down." {
		t.Errorf("a refusal is said in plain words, not a program's error: %q", refusal.Error)
	}

	// Every upload carries Record, hidden until the page can record.
	if list := get(t, h, "/t/file").Body.String(); !strings.Contains(list, `data-component="voice" data-mode="record" data-target="upload"`) || !strings.Contains(list, `id="voice-upload" hidden>`) {
		t.Error("the files page's upload carries Record, hidden until its script finds a microphone")
	}
}
