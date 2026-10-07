package cli

import (
	"os"
	"os/exec"
	"syscall"
)

func init() { goApart = startApart }

// startApart starts Sameway with no console, the kept program when there is
// one (keep.go), and waits for it.
func startApart(c *ctx, dir string) (bool, error) {
	if os.Getenv(apartEnv) != "" {
		return false, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return false, nil
	}
	if kept := keptProgram(); kept != "" {
		exe = kept
	}
	logPath := apartLog()
	f, err := os.Create(logPath)
	if err != nil {
		return false, nil
	}
	defer f.Close()
	cmd := exec.Command(exe, "open", "--workspace", dir)
	cmd.Env = append(os.Environ(), apartEnv+"=1")
	cmd.Stdout, cmd.Stderr = f, f
	const detached, newGroup = 0x00000008, 0x00000200
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detached | newGroup, HideWindow: true}
	if err := cmd.Start(); err != nil {
		return false, nil
	}
	return true, c.waitApart(cmd, dir, logPath)
}
