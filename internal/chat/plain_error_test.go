package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// A program's error reaches a person as what it said about the trouble:
// not its name, its exit status or a file's path, and never cut short of
// what to do.
func TestAProgramsErrorInAPersonsWords(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"claude: exit status 1: Invalid API key · Please run /login": "Invalid API key · Please run /login",
		"exec: exit status 2": "The assistant could not reach the model. Try again.",
		"open /tmp/sameway/abc.json: the model server is not running": "open: the model server is not running",
		`claude: could not read C:\Users\me\file.txt`:                 "could not read",
		"anthropic: 401 Unauthorized: invalid x-api-key":              "401 Unauthorized: invalid x-api-key",
		"": "",
		// The model the person chose is named: it is what they sign in to.
		"Claude Code on this computer is signed out. Sign in to it again, or choose another model where the page asks for one.":                            "Claude Code on this computer is signed out. Sign in to it again, or choose another model where the page asks for one.",
		"The AI model at http://127.0.0.1:11434/v1 isn't answering: it may not be running. Start it, or choose another model where the page asks for one.": "The AI model at http://127.0.0.1:11434/v1 isn't answering: it may not be running. Start it, or choose another model where the page asks for one.",
	} {
		if got := chat.SanitizeError(in); got != want {
			t.Errorf("SanitizeError(%q) = %q, want %q", in, got, want)
		}
	}
}
