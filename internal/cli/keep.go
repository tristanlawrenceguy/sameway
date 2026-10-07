package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// A person who double-clicked Sameway in their Downloads folder had to find
// it there again the next day, and lost it when they tidied Downloads. On
// Windows the first double-click now keeps the program where a person's own
// programs go (LOCALAPPDATA\Programs\Sameway) and puts it in the Start menu,
// once. A Mac keeps it in Applications, where the person drags it; a
// program run from the temporary folder (go run, a test) is never kept.

// keepProgram keeps the running program and says so, or does nothing.
var keepProgram = func(out io.Writer) {
	if runtime.GOOS != "windows" {
		return
	}
	exe, err := os.Executable()
	if err != nil || strings.HasPrefix(strings.ToLower(exe), strings.ToLower(os.TempDir())) {
		return
	}
	local, roaming := os.Getenv("LOCALAPPDATA"), os.Getenv("APPDATA")
	if local == "" || roaming == "" {
		return
	}
	lnk := filepath.Join(roaming, "Microsoft", "Windows", "Start Menu", "Programs", "Sameway.lnk")
	kept := filepath.Join(local, "Programs", "Sameway", "Sameway.exe")
	_, err = os.Stat(kept)
	first := err != nil
	// A Sameway double-clicked from elsewhere, a newer download, is the one
	// kept: the Start menu and Sameway in the background run what was
	// clicked last. One kept that is running stays as it is.
	if !strings.EqualFold(filepath.Clean(exe), kept) {
		if err := copyProgram(exe, kept); err != nil && first {
			return
		}
	}
	if _, err := os.Stat(lnk); err == nil || !first {
		return // made before; a person who took it away keeps it away
	}
	if err := shortcut(lnk, kept); err != nil {
		return
	}
	fmt.Fprintln(out, "Sameway is in your Start menu now, so you can find it there next time.")
}

func copyProgram(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.WriteFile(to, data, 0o755)
}

// shortcut makes a Start menu entry through Windows' own Shell object; Go
// has no way of its own to write a .lnk.
var shortcut = func(lnk, target string) error {
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := "$s=(New-Object -ComObject WScript.Shell).CreateShortcut(" + q(lnk) + ");$s.TargetPath=" + q(target) +
		";$s.WorkingDirectory=" + q(filepath.Dir(target)) + ";$s.Description='Sameway';$s.Save()"
	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Run()
}

// keptProgram is the program kept for the Start menu, or "" when there is
// none.
func keptProgram() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return ""
	}
	kept := filepath.Join(local, "Programs", "Sameway", "Sameway.exe")
	if _, err := os.Stat(kept); err != nil {
		return ""
	}
	return kept
}
