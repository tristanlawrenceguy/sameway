package media_test

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/speech"
)

// A running workspace writes recordings down itself, with no page open:
// one that arrives, and one already there when it starts; and a page does
// not start what the host will do.
func TestRecordingsAreWrittenDownWithNoPageOpen(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	srv := h.(*server.Server)
	srv.UseSpeech(server.Speech{
		Ready: func() bool { return true },
		Transcribe: func(ctx context.Context, wav string) ([]convert.Cue, error) {
			return []convert.Cue{{Start: 0, End: 1, Text: "Heard with no page."}}, nil
		},
	})
	upload := func(name string, content []byte) string {
		body, ct := multipartFile(t, name, string(content), nil)
		res := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
		return strings.TrimPrefix(res.Header().Get("Location"), "/t/file/")
	}
	var wav bytes.Buffer
	speech.WriteWAV(&wav, make([]float32, speech.Rate))
	before := upload("Before.wav", wav.Bytes())
	heard := func(id string) func() bool {
		return func() bool {
			f, _ := a.Store.Get("file", id)
			s, _ := f.Fields["text"].(string)
			return strings.Contains(s, "Heard with no page.")
		}
	}
	if heard(before)() {
		t.Fatal("nothing is written down in the background until it is on")
	}
	if page := get(t, h, "/t/file/"+before).Body.String(); !strings.Contains(page, "data-auto") {
		t.Error("until then, the page writes it down itself")
	}

	srv.WriteDownInBackground()
	waitFor(t, heard(before))
	after := upload("After.wav", wav.Bytes())
	waitFor(t, heard(after))

	memo := upload("Memo.m4a", []byte("not really audio"))
	if page := get(t, h, "/t/file/"+memo).Body.String(); strings.Contains(page, "data-auto") {
		t.Error("the page leaves to the host what the host will do")
	}
}
