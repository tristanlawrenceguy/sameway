package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/design"
)

// The Mac and Linux halves of keeping Sameway where the computer finds it
// (keep.go has Windows'). On Linux a program double-clicked in a file
// manager often runs in no terminal at all, or not at all, and nothing
// put Sameway in the applications menu: the first run adds it there, with
// its icon, once. On a Mac an app opened from Downloads runs from a copy
// macOS makes somewhere hidden, and was found again only by the person
// who dragged it to Applications: the first run copies it to the
// Applications folder of their own, once.

// keepOnLinux adds Sameway to this person's applications menu.
func keepOnLinux(out io.Writer, exe, home string) {
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(home, ".local", "share")
	}
	entry := filepath.Join(data, "applications", "sameway.desktop")
	if _, err := os.Stat(entry); err == nil {
		return // made before; a person who took it away keeps it away
	}
	icon := filepath.Join(data, "icons", "hicolor", "256x256", "apps", "sameway.png")
	if png, err := design.FS.ReadFile("brand/icon.png"); err == nil {
		if os.MkdirAll(filepath.Dir(icon), 0o755) == nil {
			os.WriteFile(icon, png, 0o644)
		}
	}
	text := "[Desktop Entry]\nType=Application\nName=Sameway\nComment=Your workspace, with an assistant\nExec=\"" +
		strings.ReplaceAll(exe, `"`, `\"`) + "\"\nIcon=sameway\nTerminal=false\nCategories=Office;\n"
	if os.MkdirAll(filepath.Dir(entry), 0o755) != nil || os.WriteFile(entry, []byte(text), 0o644) != nil {
		return
	}
	fmt.Fprintln(out, "Sameway is in your applications menu now, so you can find it there next time.")
}

// keepOnMac copies Sameway.app to the person's own Applications folder when
// it runs from anywhere else, and says whether it did.
func keepOnMac(out io.Writer, exe, home string) bool {
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe))) // Sameway.app/Contents/MacOS/sameway
	if filepath.Ext(bundle) != ".app" {
		return false
	}
	for _, apps := range []string{"/Applications", filepath.Join(home, "Applications")} {
		if strings.HasPrefix(bundle, apps+string(filepath.Separator)) {
			return false
		}
		if _, err := os.Stat(filepath.Join(apps, "Sameway.app")); err == nil {
			return false // kept before
		}
	}
	dest := filepath.Join(home, "Applications", "Sameway.app")
	if os.MkdirAll(filepath.Dir(dest), 0o755) != nil {
		return false
	}
	// ditto copies an app as the Finder does, its signature with it.
	if err := copyApp(bundle, dest); err != nil {
		return false
	}
	fmt.Fprintln(out, "Sameway is in your Applications folder now, so you can find it there next time.")
	return true
}

// copyApp copies an app bundle; tests put their own here.
var copyApp = func(from, to string) error { return exec.Command("ditto", from, to).Run() }
