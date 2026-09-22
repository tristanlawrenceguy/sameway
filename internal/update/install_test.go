package update_test

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// binary is a file standing in for the running program, so an install has
// something to replace.
func binary(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sameway")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInstallPutsTheNewProgramWhereTheOldOneWas(t *testing.T) {
	srv := releases(t, release{version: "0.4.0", body: []byte("new program")})
	exe := binary(t, "old program")
	out, err := update.Updater{Current: "0.3.0", Repo: "o/r", API: srv.URL, Exe: exe}.Run(context.Background(), true)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if !out.Installed || out.Path != exe {
		t.Fatalf("installed=%v path=%q, want true and %q", out.Installed, out.Path, exe)
	}
	if !strings.Contains(out.Says, "0.4.0 is installed") || !strings.Contains(out.Says, "next start") {
		t.Errorf("says %q; it has to say the new version runs from the next start", out.Says)
	}
	got, err := os.ReadFile(exe)
	if err != nil || string(got) != "new program" {
		t.Fatalf("the program was not replaced: %q %v", got, err)
	}
	// Nothing is left beside it for the next start to trip over, except on
	// Windows where the old program cannot be deleted while it runs.
	if _, err := os.Stat(exe + ".new"); err == nil {
		t.Error("the download was left behind")
	}
}

func TestInstallRefusesADownloadThatDoesNotMatchItsSum(t *testing.T) {
	srv := releases(t, release{version: "0.4.0", body: []byte("tampered"), sums: []byte("0000000000000000000000000000000000000000000000000000000000000000  sameway\n")})
	exe := binary(t, "old program")
	_, err := update.Updater{Current: "0.3.0", Repo: "o/r", API: srv.URL, Exe: exe}.Run(context.Background(), true)
	if err == nil {
		t.Fatal("a download that is not in checksums.txt was installed")
	}
	if !strings.Contains(err.Error(), "checksums.txt") {
		t.Errorf("the error should name checksums.txt: %v", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old program" {
		t.Errorf("the program was replaced anyway: %q", got)
	}
}

func TestInstallRefusesASumThatDisagrees(t *testing.T) {
	name := update.AssetName("0.4.0")
	srv := releases(t, release{version: "0.4.0", body: []byte("tampered"),
		sums: []byte("0000000000000000000000000000000000000000000000000000000000000000  " + name + "\n")})
	exe := binary(t, "old program")
	_, err := update.Updater{Current: "0.3.0", Repo: "o/r", API: srv.URL, Exe: exe}.Run(context.Background(), true)
	if err == nil || !strings.Contains(err.Error(), "did not match") {
		t.Fatalf("expected a mismatch to stop the install, got %v", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old program" {
		t.Errorf("the program was replaced anyway: %q", got)
	}
}

func TestInstallRefusesARelaseWithNoChecksumsAtAll(t *testing.T) {
	srv := releases(t, release{version: "0.4.0", noSums: true})
	exe := binary(t, "old program")
	_, err := update.Updater{Current: "0.3.0", Repo: "o/r", API: srv.URL, Exe: exe}.Run(context.Background(), true)
	if err == nil || !strings.Contains(err.Error(), "cannot be checked") {
		t.Fatalf("expected a release without checksums to be refused, got %v", err)
	}
}

func TestADevBuildNeverReplacesItself(t *testing.T) {
	srv := releases(t, release{version: "0.4.0"})
	exe := binary(t, "built from source")
	out, err := update.Updater{Current: "dev", Repo: "o/r", API: srv.URL, Exe: exe}.Run(context.Background(), true)
	if err != nil {
		t.Fatalf("a dev build asking to update is not an error: %v", err)
	}
	if out.Installed {
		t.Fatal("a dev build replaced itself")
	}
	if !strings.Contains(out.Says, "0.4.0") || !strings.Contains(out.Says, "will not replace itself") {
		t.Errorf("says %q; it has to name the latest release and say why it stopped", out.Says)
	}
	if got, _ := os.ReadFile(exe); string(got) != "built from source" {
		t.Errorf("the program was replaced anyway: %q", got)
	}
}

func TestInstallTakesTheProgramOutOfAnArchive(t *testing.T) {
	program := []byte("new program")
	for _, c := range []struct {
		name string
		body []byte
	}{
		{"sameway_0.4.0_" + goos() + ".zip", zipped(t, program)},
		{"sameway_0.4.0_" + goos() + ".tar.gz", tarred(t, program)},
	} {
		srv := releases(t, release{version: "0.4.0", name: c.name, body: c.body})
		exe := binary(t, "old program")
		out, err := update.Updater{Current: "0.3.0", Repo: "o/r", API: srv.URL, Exe: exe}.Run(context.Background(), true)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !out.Installed {
			t.Fatalf("%s: nothing was installed", c.name)
		}
		if got, _ := os.ReadFile(exe); string(got) != string(program) {
			t.Errorf("%s: the program inside the archive did not land: %q", c.name, got)
		}
	}
}

func TestTidyClearsWhatAnInstallLeftBehind(t *testing.T) {
	exe := binary(t, "program")
	for _, suffix := range []string{".old", ".new"} {
		if err := os.WriteFile(exe+suffix, []byte("leftover"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	update.Tidy(exe)
	for _, suffix := range []string{".old", ".new"} {
		if _, err := os.Stat(exe + suffix); err == nil {
			t.Errorf("%s was left behind", exe+suffix)
		}
	}
	if _, err := os.Stat(exe); err != nil {
		t.Errorf("Tidy removed the program itself: %v", err)
	}
}

func TestWatchInstallsInAutoAndOnlyTellsInManual(t *testing.T) {
	for _, c := range []struct {
		mode    string
		install bool
	}{{update.Auto, true}, {update.Manual, false}} {
		srv := releases(t, release{version: "0.4.0", body: []byte("new program")})
		exe := binary(t, "old program")
		ctx, stop := context.WithCancel(context.Background())
		told := make(chan update.Outcome, 4)
		update.Updater{Current: "0.3.0", Repo: "o/r", API: srv.URL, Exe: exe}.Watch(ctx, func() string { return c.mode }, time.Hour, func(o update.Outcome, err error) {
			if err != nil {
				t.Errorf("%s: %v", c.mode, err)
			}
			told <- o
		})
		var out update.Outcome
		select {
		case out = <-told:
		case <-time.After(10 * time.Second):
			t.Fatalf("%s: nothing was said about the new version", c.mode)
		}
		stop()
		if out.Latest != "0.4.0" || out.Installed != c.install {
			t.Errorf("%s: latest=%q installed=%v, want 0.4.0 and %v", c.mode, out.Latest, out.Installed, c.install)
		}
		body, _ := os.ReadFile(exe)
		if want := map[bool]string{true: "new program", false: "old program"}[c.install]; string(body) != want {
			t.Errorf("%s: the program is %q, want %q", c.mode, body, want)
		}
		if !c.install && !strings.Contains(out.Says, "sameway update") {
			t.Errorf("manual: says %q; it has to say how to get the new version", out.Says)
		}
	}
}

func TestADevBuildDoesNotWatch(t *testing.T) {
	srv := releases(t, release{version: "0.4.0"})
	told := make(chan update.Outcome, 1)
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	update.Updater{Current: "dev", Repo: "o/r", API: srv.URL, Exe: binary(t, "x")}.Watch(ctx, func() string { return update.Auto }, time.Hour, func(o update.Outcome, err error) { told <- o })
	select {
	case o := <-told:
		t.Errorf("a dev build watched and said %q", o.Says)
	case <-time.After(300 * time.Millisecond):
	}
}

// goos names this machine the way a release file does, so the archive
// tests publish something the updater will pick up.
func goos() string {
	return runtime.GOOS + "_" + runtime.GOARCH
}

func zipped(t *testing.T, program []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("sameway")
	if err != nil {
		t.Fatal(err)
	}
	w.Write(program)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func tarred(t *testing.T, program []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "sameway", Mode: 0o755, Size: int64(len(program)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	tw.Write(program)
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
