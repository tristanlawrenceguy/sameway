package speech

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The engine's lines of what was heard become cues, and cues a WebVTT.
func TestTheEnginesLinesBecomeCues(t *testing.T) {
	t.Parallel()
	out := "Started\nnum threads: 4\n0.326 -- 16.704: God as a direct consequence\n17.000 -- 18.500:  \n18.5 -- 20.25: And then.\nElapsed seconds: 1.5 s\n"
	cues := Parse(out)
	if len(cues) != 2 || cues[0].Start != 0.326 || cues[0].End != 16.704 || cues[0].Text != "God as a direct consequence" || cues[1].Text != "And then." {
		t.Fatalf("two cues, got %+v", cues)
	}
	vtt := VTT(cues)
	if !strings.HasPrefix(vtt, "WEBVTT\n\n00:00:00.326 --> 00:00:16.704\nGod as") || !strings.Contains(vtt, "00:00:18.500 --> 00:00:20.250\nAnd then.") {
		t.Errorf("a WebVTT with a cue each: %q", vtt)
	}
}

// Any PCM WAV, at any rate and in stereo, becomes the engine's 16 kHz mono.
func TestAWAVBecomesTheSoundTheEngineReads(t *testing.T) {
	t.Parallel()
	// A 48 kHz stereo 16-bit WAV of one second, left at half, right silent,
	// with a chunk before its sound that is passed over.
	var in bytes.Buffer
	frames := 48000
	in.WriteString("RIFF\x00\x00\x00\x00WAVEfmt \x10\x00\x00\x00\x01\x00\x02\x00\x80\xbb\x00\x00\x00\xee\x02\x00\x04\x00\x10\x00LIST\x02\x00\x00\x00xxdata")
	size := frames * 4
	in.Write([]byte{byte(size), byte(size >> 8), byte(size >> 16), byte(size >> 24)})
	for i := 0; i < frames; i++ {
		in.Write([]byte{0x00, 0x40, 0x00, 0x00})
	}
	samples, rate, err := ReadWAV(&in)
	if err != nil || rate != 48000 || len(samples) != frames {
		t.Fatalf("read %d samples at %d: %v", len(samples), rate, err)
	}
	if samples[10] < 0.24 || samples[10] > 0.26 {
		t.Errorf("channels are averaged: %v", samples[10])
	}
	out := Resample(samples, rate)
	if len(out) != 16000 {
		t.Errorf("a second at 16 kHz is 16000 samples, got %d", len(out))
	}
	var w bytes.Buffer
	if err := WriteWAV(&w, out); err != nil {
		t.Fatal(err)
	}
	back, rate2, err := ReadWAV(&w)
	if err != nil || rate2 != 16000 || len(back) != 16000 {
		t.Errorf("what is written reads back: %d at %d, %v", len(back), rate2, err)
	}
	if _, _, err := ReadWAV(strings.NewReader("ID3 an mp3")); err == nil {
		t.Error("a file that is not a WAV is said to be not one")
	}
}

// Getting speech-to-text fetches each file, keeps it only when its
// fingerprint is the one expected, unpacks the engine, and fetches nothing
// again that is already here.
func TestSpeechToTextIsFetchedCheckedAndKept(t *testing.T) {
	t.Parallel()
	exe := "sherpa-onnx-vad-with-offline-asr"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	var arc bytes.Buffer
	gz := gzip.NewWriter(&arc)
	tw := tar.NewWriter(gz)
	for _, f := range []struct{ name, body string }{{"pkg/bin/" + exe, "program"}, {"pkg/lib/libx.so", "lib"}} {
		tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o755, Size: int64(len(f.body)), Typeflag: tar.TypeReg})
		tw.Write([]byte(f.body))
	}
	tw.Close()
	gz.Close()
	files := map[string][]byte{"/engine.tar.gz": arc.Bytes(), "/enc": []byte("encoder"), "/dec": []byte("decoder"), "/tok": []byte("tokens"), "/vad": []byte("vad")}
	fetched := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetched[r.URL.Path]++
		w.Write(files[r.URL.Path])
	}))
	defer srv.Close()
	sum := func(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
	a := func(p, name string) asset {
		return asset{URL: srv.URL + p, SHA256: sum(files[p]), Size: int64(len(files[p])), Name: name}
	}
	eng := a("/engine.tar.gz", "engine.tar.gz")
	mod := []asset{a("/enc", "encoder.onnx"), a("/dec", "decoder.onnx"), a("/tok", "tokens.txt"), a("/vad", "vad.onnx")}
	dir := t.TempDir()

	bad := append([]asset{}, mod...)
	bad[1].SHA256 = strings.Repeat("0", 64)
	if err := install(context.Background(), srv.Client(), dir, eng, bad, nil); err == nil || !strings.Contains(err.Error(), "not the file sameway expects") {
		t.Fatalf("a file with another fingerprint is refused: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "decoder.onnx")); err == nil {
		t.Error("and not kept")
	}

	var last int64
	if err := install(context.Background(), srv.Client(), dir, eng, mod, func(done, total int64) { last = done }); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(runner(dir)); string(b) != "program" {
		t.Error("the engine is unpacked, its top folder dropped")
	}
	if _, err := os.Stat(filepath.Join(dir, "engine.tar.gz")); err == nil {
		t.Error("the archive goes once it is unpacked")
	}
	if last == 0 {
		t.Error("progress is told")
	}
	before := fetched["/enc"] + fetched["/engine.tar.gz"]
	if err := install(context.Background(), srv.Client(), dir, eng, mod, nil); err != nil {
		t.Fatal(err)
	}
	if fetched["/enc"]+fetched["/engine.tar.gz"] != before {
		t.Error("nothing already here is fetched again")
	}
}

// An archive entry that would land outside the engine's folder is refused.
func TestAnArchiveCannotWriteOutsideItself(t *testing.T) {
	t.Parallel()
	var arc bytes.Buffer
	gz := gzip.NewWriter(&arc)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "pkg/../../evil", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg})
	tw.Write([]byte("x"))
	tw.Close()
	gz.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "a.tar.gz")
	os.WriteFile(path, arc.Bytes(), 0o644)
	if err := unpack(path, filepath.Join(dir, "out")); err == nil {
		t.Error("an entry outside the archive is refused")
	}
}

// Every system sameway is built for has an engine pinned by fingerprint.
func TestEverySystemHasAPinnedEngine(t *testing.T) {
	t.Parallel()
	for _, sys := range []string{"windows/amd64", "windows/arm64", "darwin/arm64", "darwin/amd64", "linux/amd64", "linux/arm64"} {
		e, ok := engines[sys]
		if !ok || len(e.SHA256) != 64 || !strings.HasPrefix(e.URL, "https://github.com/k2-fsa/sherpa-onnx/releases/download/v"+EngineVersion+"/") {
			t.Errorf("%s has no pinned engine: %+v", sys, e)
		}
	}
	for _, m := range model {
		if len(m.SHA256) != 64 || m.Size == 0 {
			t.Errorf("model file %s is not pinned", m.Name)
		}
	}
}

// The program's lines are read as turns, the voices numbered in the order
// they are first heard.
func TestTurnsAreNumberedAsTheyAreHeard(t *testing.T) {
	t.Parallel()
	out := "Started\n0.031 -- 2.039 speaker_01\n2.039 -- 3.777 speaker_00\n5.330 -- 5.971 speaker_01\nprogress 100.00%\n"
	turns := ParseTurns(out)
	if len(turns) != 3 || turns[0].Speaker != 1 || turns[1].Speaker != 2 || turns[2].Speaker != 1 || turns[1].Start != 2.039 {
		t.Errorf("turns: %+v", turns)
	}
}
