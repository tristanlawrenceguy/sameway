package llm

import (
	"os/exec"
	"syscall"
)

// apart starts a program with no window of its own: Ollama's server would
// otherwise open a console beside Sameway's.
func apart(cmd *exec.Cmd) {
	const noWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: noWindow, HideWindow: true}
}
