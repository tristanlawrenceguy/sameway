//go:build !windows

package llm

import "os/exec"

// apart is Windows' alone: elsewhere a server started opens no window.
func apart(cmd *exec.Cmd) {}
