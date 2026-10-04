package chat

import (
	"regexp"
	"strings"
)

var (
	// "claude: ", "anthropic: ", "exec: " at the start of a program's error.
	errProgram = regexp.MustCompile(`^(?:[a-z][a-z0-9_-]*: )+`)
	// "exit status 1", which says only that it failed.
	errExit = regexp.MustCompile(`(?i)\bexit status \d+:?\s*`)
	// A file's path, two parts or more, or one on a Windows drive: never a
	// slash command such as /login, which tells the person what to do.
	errPath = regexp.MustCompile(`(?:[A-Za-z]:\\[^\s]+|(?:/[\w.\-]+){2,}/?)`)
)

// SanitizeError is a program's error in a person's words: what the
// program said about the trouble, without its name, its exit status or a
// file's path, which say nothing to them. "claude: exit status 1: Invalid
// API key · Please run /login" is "Invalid API key · Please run /login".
// With nothing left, it says the model did not answer.
func SanitizeError(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	for {
		next := strings.TrimSpace(errExit.ReplaceAllString(errProgram.ReplaceAllString(s, ""), ""))
		if next == s {
			break
		}
		s = next
	}
	s = strings.Join(strings.Fields(errPath.ReplaceAllString(s, "")), " ")
	s = strings.Trim(strings.ReplaceAll(s, " :", ":"), " :;,")
	// What the system says of a failed connection or file tells a person
	// nothing they can act on; the plain sentence below does.
	for _, phrase := range []string{"connection refused", "no such file or directory", "timed out", "network is unreachable", "permission denied", "i/o timeout"} {
		if strings.Contains(strings.ToLower(s), phrase) {
			s = ""
		}
	}
	if s == "" {
		return "The assistant could not reach the model. Try again."
	}
	return s
}
