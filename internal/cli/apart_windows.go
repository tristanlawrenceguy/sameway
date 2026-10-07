package cli

import (
	"os"
	"os/exec"
	"syscall"
)

func init() {
	goApart = startApart
	atLogin = func(c *ctx, dir string) error {
		_, _, err := spawnApart("open", "--workspace", dir, "--no-browser")
		return err
	}
}

// startApart starts Sameway with no console and waits for it.
func startApart(c *ctx, dir string) (bool, error) {
	if os.Getenv(apartEnv) != "" {
		return false, nil
	}
	cmd, logPath, err := spawnApart("open", "--workspace", dir)
	if err != nil {
		return false, nil
	}
	return true, c.waitApart(cmd, dir, logPath)
}

// spawnApart starts this program, the kept one when there is one
// (keep.go), with no console and outside this one's, writing what it says
// to the background log.
func spawnApart(args ...string) (*exec.Cmd, string, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, "", err
	}
	if kept := keptProgram(); kept != "" {
		exe = kept
	}
	logPath := apartLog()
	f, err := os.Create(logPath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), apartEnv+"=1")
	cmd.Stdout, cmd.Stderr = f, f
	const detached, newGroup = 0x00000008, 0x00000200
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detached | newGroup, HideWindow: true}
	if err := cmd.Start(); err != nil {
		return nil, "", err
	}
	return cmd, logPath, nil
}
