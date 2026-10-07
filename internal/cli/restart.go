package cli

import (
	"net"
	"os"
	"os/exec"
	"time"
)

// An update is installed beside the running program and runs from its
// next start. Restart Sameway, offered on the page once one is waiting
// (internal/server restart.go), is that start: this program is started
// again on the same workspace and address, in the background with no
// browser tab, and this one stops. The new one waits for the address while
// the old one lets it go, and the page, waiting too, carries on.

// restartEnv marks a Sameway started by a restart, which waits for its
// address.
const restartEnv = "SAMEWAY_RESTART"

// startAgain starts the program as it is on disk now on dir and addr.
func startAgain(dir, addr string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "open", "--workspace", dir, "--addr", addr, "--no-browser")
	cmd.Env = append(os.Environ(), restartEnv+"=1", apartEnv+"=1")
	f, err := os.Create(apartLog())
	if err == nil {
		defer f.Close()
		cmd.Stdout, cmd.Stderr = f, f
	}
	detach(cmd) // restart_windows.go, restart_other.go
	return cmd.Start()
}

// listenFor listens on addr; a Sameway started by a restart waits up to
// half a minute for the one before it to let the address go.
func listenFor(addr string) (net.Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err == nil || os.Getenv(restartEnv) == "" {
		return l, err
	}
	for deadline := time.Now().Add(30 * time.Second); time.Now().Before(deadline); {
		time.Sleep(250 * time.Millisecond)
		if l, err = net.Listen("tcp", addr); err == nil {
			return l, nil
		}
	}
	return nil, err
}
