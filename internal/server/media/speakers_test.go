package media_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
)

// With speakers told apart here, a recording written down whole has each
// line said by Speaker 1, Speaker 2, in the order they are first heard.
func TestARecordingSaysWhoSpoke(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	srv := h.(*server.Server)
	srv.UseSpeech(server.Speech{
		Ready: func() bool { return true },
		Transcribe: func(ctx context.Context, wav string) ([]convert.Cue, error) {
			return []convert.Cue{{Start: 0.2, End: 2, Text: "Morning, two things."}, {Start: 2.2, End: 4, Text: "Go on."}, {Start: 4.2, End: 6, Text: "The liner is late."}}, nil
		},
	})
	srv.UseSpeakers(func() bool { return true }, func(ctx context.Context, wav string, n int) ([]speech.Turn, error) {
		return []speech.Turn{{Start: 0, End: 2.1, Speaker: 1}, {Start: 2.1, End: 4.1, Speaker: 2}, {Start: 4.1, End: 6, Speaker: 1}}, nil
	})
	body, ct := multipartFile(t, "Memo.m4a", "not really audio", nil)
	id := strings.TrimPrefix(do(t, h, http.MethodPost, "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
	var wav bytes.Buffer
	speech.WriteWAV(&wav, make([]float32, 8000))
	req := httptest.NewRequest(http.MethodPost, "/files/"+id+"/transcribe", &wav)
	req.Header.Set("Content-Type", "audio/wav")
	h.ServeHTTP(httptest.NewRecorder(), req)
	text := ""
	for i := 0; i < 300 && !strings.Contains(text, "late"); i++ {
		f, _ := a.Store.Get("file", id)
		text, _ = f.Fields["text"].(string)
		time.Sleep(10 * time.Millisecond)
	}
	for _, want := range []string{"**Speaker 1:** Morning, two things.", "**Speaker 2:** Go on.", "**Speaker 1:** The liner is late."} {
		if !strings.Contains(text, want) {
			t.Errorf("the transcript should say %q: %q", want, text)
		}
	}
}
