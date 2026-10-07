package cli

import (
	"os/exec"
	"syscall"
)

// detach starts a program with no console and outside this one's, so it
// lives on when this one stops.
func detach(cmd *exec.Cmd) {
	const detached, newGroup = 0x00000008, 0x00000200
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detached | newGroup, HideWindow: true}
}
