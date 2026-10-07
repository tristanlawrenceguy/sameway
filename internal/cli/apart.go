package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A double-click on Windows opened a black window that had to stay open
// for as long as Sameway was used, and closing it, by habit or by mistake,
// stopped Sameway. Now the double-clicked program starts Sameway apart from
// its window, with no console, waits until it answers, says where it is and
// how to stop it, and closes. If Sameway cannot start, the window stays
// with the reason. Quit Sameway on the Workspaces page stops it.

// apartEnv marks the Sameway started apart, which does not start another.
const apartEnv = "SAMEWAY_APART"

// goApart starts Sameway on dir apart from this window and says whether it
// did; false means start it here, as before. Windows' alone (apart_windows.go).
var goApart = func(c *ctx, dir string) (bool, error) { return false, nil }

// atLogin opens a workspace as the computer starts: apart and at once on
// Windows, where it is run from a minimised window that should go; here
// otherwise, with no browser (internal/atlogin).
var atLogin = func(c *ctx, dir string) error {
	c.workspaceDir, c.args = dir, []string{"--no-browser"}
	return c.openCmd()
}

// atLoginCmd is sameway at-login, what the sign-in entry runs.
func (c *ctx) atLoginCmd() error {
	dir := c.workspaceDir
	if dir == "" {
		var err error
		if dir, _, err = c.yourWorkspace(); err != nil {
			return err
		}
	}
	return atLogin(c, dir)
}

// waitApart waits for the Sameway started apart to answer for dir, or to
// end, and says which.
func (c *ctx) waitApart(cmd *exec.Cmd, dir, logPath string) error {
	ended := make(chan error, 1)
	go func() { ended <- cmd.Wait() }()
	deadline := time.Now().Add(time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ended:
			return fmt.Errorf("Sameway stopped as it started:\n%s", tail(logPath, 12))
		case <-time.After(500 * time.Millisecond):
		}
		for _, k := range workspace.KnownWorkspaces() {
			if sameDir(k.Dir, dir) && k.Addr != "" && answers("http://"+k.Addr) {
				fmt.Fprintf(c.Stdout, "Sameway is open at http://%s/\nIt runs in the background, so this window can close.\nTo stop it, choose Quit Sameway on the Workspaces page.\n", k.Addr)
				time.Sleep(4 * time.Second)
				return nil
			}
		}
	}
	return fmt.Errorf("Sameway did not answer within a minute. What it said is in %s", logPath)
}

// tail is the last n lines of a file, for an error the person reads.
func tail(path string, n int) string {
	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func sameDir(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

// apartLog is where the Sameway started apart writes what a terminal would
// have shown: beside the person's keys, never in the workspace.
func apartLog() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	dir = filepath.Join(dir, "sameway")
	os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "background.log")
}
