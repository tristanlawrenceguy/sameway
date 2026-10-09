package server_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
)

// A call recorded with the computer's sound says who was heard each
// second; written down, each line is said by me or by them, and them is
// the other person by name when the meeting has one.
func TestACallIsWrittenDownAsMeAndThem(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	h.(*server.Server).UseSpeech(server.Speech{
		Ready: func() bool { return true },
		Transcribe: func(ctx context.Context, wav string) ([]convert.Cue, error) {
			return []convert.Cue{{Start: 0.3, End: 2.1, Text: "Order the compost."}, {Start: 4, End: 6, Text: "I will, today."}}, nil
		},
	})
	ann, _ := a.Store.Create("person", map[string]any{"name": "Ann Lee"})
	ev, _ := a.Store.Create("event", map[string]any{"title": "Call with Ann", "people": []any{ann.ID}})

	body, ct := multipartFile(t, "Recording.m4a", "not really audio", url.Values{"voices": {"mmm.ttttt"}})
	do(t, h, http.MethodPost, "/t/file/upload?meeting="+ev.ID, body, ct)
	got, _ := a.Store.Get("event", ev.ID)
	id, _ := got.Fields["recording"].(string)
	if id == "" {
		t.Fatal("the call is the meeting's recording")
	}
	var wav bytes.Buffer
	speech.WriteWAV(&wav, make([]float32, 8000))
	req := httptest.NewRequest(http.MethodPost, "/files/"+id+"/transcribe", &wav)
	req.Header.Set("Content-Type", "audio/wav")
	h.ServeHTTP(httptest.NewRecorder(), req)
	text := ""
	for i := 0; i < 300 && !strings.Contains(text, "today"); i++ {
		f, _ := a.Store.Get("file", id)
		text, _ = f.Fields["text"].(string)
		time.Sleep(10 * time.Millisecond)
	}
	if !strings.Contains(text, "**Me:** Order the compost.") || !strings.Contains(text, "**Ann Lee:** I will, today.") {
		t.Errorf("each line is said by me or by them, by name: %q", text)
	}
}
