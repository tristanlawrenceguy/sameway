package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
)

// A long recording is written down a part at a time: the host says how
// (here for a WAV, in chunks it copied out, or whole), each part is
// written down in turn at its place, the page is told how far it has
// come, and the parts are put together at their times.
func TestALongRecordingIsWrittenDownInParts(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	srv := h.(*server.Server)
	var mu sync.Mutex
	heard := 0
	srv.UseSpeech(server.Speech{
		Ready: func() bool { return true },
		Transcribe: func(ctx context.Context, wav string) ([]convert.Cue, error) {
			mu.Lock()
			heard++
			n := heard
			mu.Unlock()
			return []convert.Cue{{Start: 1, End: 2, Text: map[int]string{1: "The first part.", 2: "The second part."}[n]}}, nil
		},
	})
	upload := func(name string, content []byte) string {
		body, ct := multipartFile(t, name, string(content), nil)
		res := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
		return strings.TrimPrefix(res.Header().Get("Location"), "/t/file/")
	}
	plan := func(id string) map[string]any {
		var p map[string]any
		json.Unmarshal(get(t, h, "/files/"+id+"/sound").Body.Bytes(), &p)
		return p
	}

	// An MP3 is copied out as chunks the page reads; a WAV is done here;
	// anything else is read whole.
	var mp3 []byte
	for i := 0; i < 40; i++ {
		f := make([]byte, 417)
		copy(f, []byte{0xFF, 0xFB, 0x90, 0x00})
		mp3 = append(mp3, f...)
	}
	talk := upload("Talk.mp3", mp3)
	p := plan(talk)
	chunks, _ := p["chunks"].([]any)
	if len(chunks) != 1 || chunks[0].(map[string]any)["url"] != "/files/"+talk+"/sound/0" {
		t.Fatalf("an MP3's sound is listed as chunks to read: %v", p)
	}
	if res := get(t, h, "/files/"+talk+"/sound/0"); res.Header().Get("Content-Type") != "audio/mpeg" || res.Body.Len() != len(mp3) {
		t.Errorf("a chunk is served as the sound it is: %q %d", res.Header().Get("Content-Type"), res.Body.Len())
	}
	if p := plan(upload("Odd.flac", []byte("fLaC not really"))); p["whole"] != true {
		t.Errorf("what cannot be copied out is read whole: %v", p)
	}

	// Two parts, sent as the page sends them, come together at their times.
	send := func(id, query string) *httptest.ResponseRecorder {
		var wav bytes.Buffer
		speech.WriteWAV(&wav, make([]float32, 1600))
		req := httptest.NewRequest(http.MethodPost, "/files/"+id+"/transcribe"+query, &wav)
		req.Header.Set("Content-Type", "audio/wav")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	wantStatus(t, send(talk, "?part=0&of=2&start=0"), http.StatusAccepted)
	waitFor(t, func() bool {
		f, _ := a.Store.Get("file", talk)
		return strings.Contains(f.Fields["note"].(string), "1 of 2 parts done")
	})
	if page := get(t, h, "/t/file/"+talk).Body.String(); !strings.Contains(page, "1 of 2 parts done") {
		t.Error("the page says how far it has come")
	}
	wantStatus(t, send(talk, "?part=1&of=2&start=600"), http.StatusAccepted)
	var text string
	waitFor(t, func() bool {
		f, _ := a.Store.Get("file", talk)
		text, _ = f.Fields["text"].(string)
		return f.Fields["status"] == "ready" && text != ""
	})
	if text != "[0:01] The first part.\n\n[10:01] The second part." {
		t.Errorf("the parts come together at their times: %q", text)
	}
	if _, err := os.Stat(a.Workspace.FilesDir() + "/" + talk + ".sound"); err == nil {
		t.Error("the copied-out sound goes once it is written down")
	}

	// A WAV is written down here, with nothing sent.
	var long bytes.Buffer
	speech.WriteWAV(&long, make([]float32, speech.Rate*2))
	memo := upload("Memo.wav", long.Bytes())
	if p := plan(memo); p["host"] != true {
		t.Errorf("a WAV is written down on the host: %v", p)
	}
	mu.Lock()
	heard = 0
	mu.Unlock()
	wantStatus(t, do(t, h, http.MethodPost, "/files/"+memo+"/transcribe", nil, ""), http.StatusSeeOther)
	waitFor(t, func() bool {
		f, _ := a.Store.Get("file", memo)
		s, _ := f.Fields["text"].(string)
		return strings.Contains(s, "The first part.")
	})
}
