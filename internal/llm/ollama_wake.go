package llm

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Ollama installed and not running (closed from the tray, not started after
// a restart) left the page saying the model at an address was not
// answering, and then to download Ollama, which the person had. Sameway now
// starts it: its server on Windows, with no window of its own (the tray
// app was tried, and with an update waiting it started no server), the app
// on a Mac, ollama serve elsewhere. A server Sameway started may stop when
// Sameway does; the next page that needs it starts it again. On Windows it
// takes some fifteen seconds to answer while it finds the graphics card.

// ollamaInstalled is how to start Ollama on this computer, or nil when it
// is not installed.
func ollamaInstalled() *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		if exe := filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Ollama", "ollama.exe"); exists(exe) {
			cmd := exec.Command(exe, "serve")
			apart(cmd)
			return cmd
		}
	case "darwin":
		if exists("/Applications/Ollama.app") {
			return exec.Command("open", "-a", "Ollama")
		}
	}
	if path, err := exec.LookPath("ollama"); err == nil {
		return exec.Command(path, "serve")
	}
	return nil
}

// Wake starts an installed Ollama that is not answering, and says whether
// it did.
func (o Ollama) Wake() bool {
	start := o.Start
	if start == nil {
		start = ollamaInstalled
	}
	cmd := start()
	if cmd == nil {
		return false
	}
	if err := cmd.Start(); err != nil {
		return false
	}
	go cmd.Wait() // reaped when it ends
	return true
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
