package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// sameway --help and -h say the commands; alone, a flag was taken for no
// command, and opened the workspace.
func TestHelpAloneSaysTheCommands(t *testing.T) {
	t.Setenv("SAMEWAY_KNOWN", filepath.Join(t.TempDir(), "known.json"))
	for _, arg := range []string{"--help", "-h", "-help"} {
		var out, errOut bytes.Buffer
		if code := Run([]string{arg}, Env{Stdout: &out, Stderr: &errOut, Dir: t.TempDir()}); code != 0 || !strings.Contains(out.String(), "Global flags:") {
			t.Errorf("%s: exit %d, said %q %q", arg, code, out.String(), errOut.String())
		}
	}
}

// A command run outside any workspace says how to name one, and names
// those this computer knows, saying which is open.
func TestOutsideAWorkspaceSaysWhichThereAre(t *testing.T) {
	t.Setenv("SAMEWAY_KNOWN", filepath.Join(t.TempDir(), "known.json"))
	running := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer running.Close()
	addr := strings.TrimPrefix(running.URL, "http://")
	known := t.TempDir()
	os.WriteFile(filepath.Join(known, workspace.ConfigFile), []byte("name: Kept\n"), 0o644)
	workspace.ThisMachine().Remember(known, addr)

	var out, errOut bytes.Buffer
	Run([]string{"describe"}, Env{Stdout: &out, Stderr: &errOut, Dir: t.TempDir()})
	said := errOut.String()
	for _, want := range []string{"--workspace <dir>", known, "open at http://" + addr} {
		if !strings.Contains(said, want) {
			t.Errorf("the error says %q: %s", want, said)
		}
	}
}
