// Package atlogin opens Sameway when the person signs in to their
// computer. Reminders ring and scheduled actions run only while Sameway
// runs, and a person who did not open it that day missed them. Each system
// has its own place for what starts at sign-in, and the entry there is the
// whole of the setting: on when it is there, off when it is not.
package atlogin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Path is this person's entry for Sameway among what starts at sign-in, or
// "" when this system has no such place Sameway knows.
func Path() string {
	switch runtime.GOOS {
	case "windows":
		if d := os.Getenv("APPDATA"); d != "" {
			return filepath.Join(d, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "Sameway.lnk")
		}
	case "darwin":
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Library", "LaunchAgents", "io.github.tristanlawrenceguy.sameway.plist")
		}
	case "linux", "freebsd":
		dir := os.Getenv("XDG_CONFIG_HOME")
		if dir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return ""
			}
			dir = filepath.Join(home, ".config")
		}
		return filepath.Join(dir, "autostart", "sameway.desktop")
	}
	return ""
}

// On says whether Sameway opens at sign-in.
func On() bool {
	p := Path()
	if p == "" {
		return false
	}
	_, err := os.Stat(p)
	return err == nil
}

// Set opens Sameway's workspace dir at sign-in with the program exe, or
// stops it. It opens without a browser tab: the person did not ask to see
// it, only for it to be there.
func Set(on bool, exe, dir string) error {
	p := Path()
	if p == "" {
		return fmt.Errorf("this computer has no place Sameway knows for what opens at sign-in")
	}
	if !on {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		// at-login starts Sameway apart and ends, so the window it opens in,
		// minimised, goes at once (internal/cli).
		return Shortcut(p, exe, `at-login --workspace "`+dir+`"`, filepath.Dir(exe), true)
	case "darwin":
		return os.WriteFile(p, []byte(plist(exe, dir)), 0o644)
	default:
		return os.WriteFile(p, []byte(desktop(exe, dir)), 0o644)
	}
}

func plist(exe, dir string) string {
	esc := func(s string) string {
		return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>io.github.tristanlawrenceguy.sameway</string>
<key>ProgramArguments</key><array><string>` + esc(exe) + `</string><string>open</string><string>--workspace</string><string>` + esc(dir) + `</string><string>--no-browser</string></array>
<key>RunAtLoad</key><true/>
</dict></plist>
`
}

func desktop(exe, dir string) string {
	q := func(s string) string { return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"` }
	return "[Desktop Entry]\nType=Application\nName=Sameway\nExec=" + q(exe) + " open --workspace " + q(dir) + " --no-browser\nX-GNOME-Autostart-enabled=true\n"
}

// Shortcut makes a Windows shortcut through Windows' own Shell object; Go
// has no way of its own to write a .lnk. Minimised opens its window in the
// taskbar only, for a program that goes at once.
var Shortcut = func(lnk, target, args, workdir string, minimised bool) error {
	q := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := "$s=(New-Object -ComObject WScript.Shell).CreateShortcut(" + q(lnk) + ");$s.TargetPath=" + q(target) +
		";$s.Arguments=" + q(args) + ";$s.WorkingDirectory=" + q(workdir) + ";$s.Description='Sameway'"
	if minimised {
		script += ";$s.WindowStyle=7"
	}
	return exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script+";$s.Save()").Run()
}
