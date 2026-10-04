package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// A program's error reaches a person as what it said about the trouble:
// not its name, its exit status or a file's path, and never cut short of
// what to do.
func TestAProgramsErrorInAPersonsWords(t *testing.T) {
	for in, want := range map[string]string{
		"claude: exit status 1: Invalid API key · Please run /login": "Invalid API key · Please run /login",
		"exec: exit status 2": "The assistant could not reach the model. Try again.",
		"open /tmp/sameway/abc.json: the model server is not running": "open: the model server is not running",
		`claude: could not read C:\Users\me\file.txt`:                 "could not read",
		"anthropic: 401 Unauthorized: invalid x-api-key":              "401 Unauthorized: invalid x-api-key",
		"": "",
	} {
		if got := chat.SanitizeError(in); got != want {
			t.Errorf("SanitizeError(%q) = %q, want %q", in, got, want)
		}
	}
}
