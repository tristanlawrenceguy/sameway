package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The first double-click on Windows keeps the program where a person's own
// programs go and puts it in the Start menu; the next one leaves it be.
func TestTheFirstDoubleClickKeepsSamewayInTheStartMenu(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the Start menu is Windows'")
	}
	local, roaming := t.TempDir(), t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("APPDATA", roaming)
	// The test binary runs from the temporary folder, which is never kept;
	// a temporary folder elsewhere stands in for Downloads.
	t.Setenv("TMP", filepath.Join(t.TempDir(), "elsewhere"))
	t.Setenv("TEMP", filepath.Join(t.TempDir(), "elsewhere"))
	var made [][2]string
	was := shortcut
	shortcut = func(lnk, target string) error {
		made = append(made, [2]string{lnk, target})
		os.MkdirAll(filepath.Dir(lnk), 0o755)
		return os.WriteFile(lnk, nil, 0o644)
	}
	defer func() { shortcut = was }()

	var out bytes.Buffer
	keepProgram(&out)
	kept := filepath.Join(local, "Programs", "Sameway", "Sameway.exe")
	if _, err := os.Stat(kept); err != nil {
		t.Fatalf("the program is kept: %v", err)
	}
	if len(made) != 1 || made[0][1] != kept || !strings.HasSuffix(made[0][0], `Start Menu\Programs\Sameway.lnk`) {
		t.Errorf("a Start menu entry leads to it: %v", made)
	}
	if !strings.Contains(out.String(), "Start menu") {
		t.Errorf("and it says so: %q", out.String())
	}
	out.Reset()
	keepProgram(&out)
	if len(made) != 1 || out.Len() != 0 {
		t.Errorf("the next double-click leaves it be: %v %q", made, out.String())
	}
}
