package server_test

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A recording is a file whose page plays it, never by itself, with its
// transcript under it on the same page, each line leading into the
// recording; until there is one, the page says so.
func TestARecordingPlaysWithItsTranscript(t *testing.T) {
	a, h := newApp(t)
	body, ct := multipartFile(t, "Stand-up.m4a", "not really audio", nil)
	res := do(t, h, http.MethodPost, "/t/file/upload", body, ct)
	wantStatus(t, res, http.StatusSeeOther)
	loc := res.Header().Get("Location")
	id := strings.TrimPrefix(loc, "/t/file/")
	file, err := a.Store.Get("file", id)
	if err != nil {
		t.Fatal(err)
	}
	if file.Fields["kind"] != "audio" || file.Fields["status"] != "ready" {
		t.Errorf("a recording is kind audio and ready: %v", file.Fields)
	}
	page := get(t, h, loc).Body.String()
	for _, want := range []string{`data-component="audio"`, `<audio class="sw-audio__media" controls preload="metadata"`, `type="audio/mp4"`, "M4A · ", "No transcript yet."} {
		if !strings.Contains(page, want) {
			t.Errorf("the recording's page should have %s", want)
		}
	}
	if strings.Contains(page, "autoplay") {
		t.Error("a recording never plays by itself")
	}
	if got := get(t, h, "/files/"+id).Header().Get("Content-Type"); got != "audio/mp4" {
		t.Errorf("an m4a is served as audio/mp4 on every system, not %q", got)
	}

	stored := file.Fields["path"].(string)
	vtt := filepath.Join(a.Workspace.FilesDir(), strings.TrimSuffix(stored, filepath.Ext(stored))+".vtt")
	if err := os.WriteFile(vtt, []byte("WEBVTT\n\n00:00:04.200 --> 00:00:09.000\n<v Hana>The compost order went in.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	page = get(t, h, loc).Body.String()
	for _, want := range []string{`<li class="sw-audio__cue" data-start="4.20">`, `href="/files/` + id + `#t=4.20"`, "Play from 4 seconds", `<span class="sw-audio__speaker">Hana:</span> The compost order went in.`} {
		if !strings.Contains(page, want) {
			t.Errorf("the transcript should have %s", want)
		}
	}
}
