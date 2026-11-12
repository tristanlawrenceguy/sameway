package cli

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// shown is the address a phone sends to the computer's screen: a page of
// this workspace (a path) on the address it is served at, or a web
// address as it came.
func shown(here, to string) string {
	if strings.HasPrefix(to, "/") {
		return here + to
	}
	return to
}

// toClipboard puts words on this computer's clipboard, the way each
// system does it, for a phone that sends them here. The words travel in
// the environment or on standard input, never in a command line.
var toClipboard = func(text string) error {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "Set-Clipboard -Value $env:SAMEWAY_CLIP")
		cmd.Env = append(os.Environ(), "SAMEWAY_CLIP="+text)
		return cmd.Run()
	case "darwin":
		return piped(text, "pbcopy")
	default:
		for _, c := range [][]string{{"wl-copy"}, {"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}} {
			if _, err := exec.LookPath(c[0]); err == nil {
				return piped(text, c[0], c[1:]...)
			}
		}
		return errors.New("this computer has no clipboard tool: install wl-clipboard or xclip")
	}
}

func piped(text, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
