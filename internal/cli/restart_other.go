//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
)

// detach starts a program in a session of its own, so it lives on when
// this one stops and the terminal it was started from closes.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
