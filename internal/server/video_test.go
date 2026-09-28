package server_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A video is a file whose page shows it with its words: served as the
// video it is on every system, a WebM judged by what is in it, never
// playing by itself; its transcript under it and on it as captions, made
// from the text, so a correction shows in both.
func TestAVideoShowsItsWordsAsCaptions(t *testing.T) {
	a, h := newApp(t)
	upload := func(name, content string) (string, map[string]any) {
		body, ct := multipartFile(t, name, content, nil)
		res := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
		wantStatus(t, res, http.StatusSeeOther)
		id := strings.TrimPrefix(res.Header().Get("Location"), "/t/file/")
		f, err := a.Store.Get("file", id)
		if err != nil {
			t.Fatal(err)
		}
		return id, f.Fields
	}

	id, talk := upload("Garden talk.mp4", "not really a video")
	if talk["kind"] != "video" {
		t.Fatalf("an MP4 is a video: %v", talk)
	}
	if got := get(t, h, "/files/"+id).Header().Get("Content-Type"); got != "video/mp4" {
		t.Errorf("served as video/mp4, not %q", got)
	}
	page := get(t, h, "/t/file/"+id).Body.String()
	for _, want := range []string{`<video class="sw-media__player" controls preload="metadata" playsinline`, `type="video/mp4"`, "No transcript yet."} {
		if !strings.Contains(page, want) {
			t.Errorf("the video's page should have %s", want)
		}
	}
	if strings.Contains(page, "autoplay") || strings.Contains(page, `<track`) {
		t.Error("a video never plays by itself, and has no captions before it has words")
	}

	stored := talk["path"].(string)
	vtt := filepath.Join(a.Workspace.FilesDir(), strings.TrimSuffix(stored, filepath.Ext(stored))+".vtt")
	os.WriteFile(vtt, []byte("WEBVTT\n\n00:00:00.000 --> 00:00:03.000\nWelcome to the garden.\n\n00:00:03.000 --> 00:00:06.000\nToday, the pond.\n"), 0o644)
	page = get(t, h, "/t/file/"+id).Body.String()
	if !strings.Contains(page, `<track kind="captions" src="/files/`+id+`/captions.vtt" label="Captions" default>`) || !strings.Contains(page, "Today, the pond.") {
		t.Error("with words, the video carries them as captions, on by default, and under it")
	}
	caps := get(t, h, "/files/"+id+"/captions.vtt")
	if caps.Header().Get("Content-Type") != "text/vtt; charset=utf-8" || !strings.Contains(caps.Body.String(), "00:00:03.000 --> 00:00:06.000\nToday, the pond.") {
		t.Errorf("the captions are WebVTT, at the times they were heard: %q", caps.Body.String())
	}
	// Corrected in the text, the captions follow: each line until the next
	// begins, the last for a few seconds.
	a.Store.Update("file", id, map[string]any{"text": "[0:00] Welcome to the garden.\n\n[0:03] Today, the fish pond."})
	if caps := get(t, h, "/files/"+id+"/captions.vtt").Body.String(); !strings.Contains(caps, "00:00:00.000 --> 00:00:03.000\nWelcome") || !strings.Contains(caps, "00:00:03.000 --> 00:00:08.000\nToday, the fish pond.") {
		t.Errorf("a correction shows in the captions: %q", caps)
	}

	// A WebM is a video when it has a picture, and a recording when not.
	webmID, clip := upload("Clip.webm", "\x1aE\xdf\xa3 ... V_VP9 ... A_OPUS")
	if clip["kind"] != "video" {
		t.Errorf("a WebM with a picture is a video: %v", clip)
	}
	if got := get(t, h, "/files/"+webmID).Header().Get("Content-Type"); got != "video/webm" {
		t.Errorf("served as video/webm, not %q", got)
	}
	if _, memo := upload("Memo.webm", "\x1aE\xdf\xa3 ... A_OPUS"); memo["kind"] != "audio" {
		t.Errorf("a WebM with only sound is a recording: %v", memo)
	}
}
