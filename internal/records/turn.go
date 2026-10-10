package records

import (
	"regexp"
	"strings"
)

// Whose turn it is in a conversation is worked out, not guessed: who wrote
// last, and whether they asked something. A small model asked the same
// missed half the questions left for the person ("Tuesday works. 3pm?");
// the question mark does not.
//
// Only the words the writer wrote this time count. A reply carries what
// it answers below it, quoted, and a question in there was asked last
// time, not now. A link's address holds a ? of its own, and a signature
// often says please ("please consider the environment"). Mail apps mark
// each of these the same few ways: lines starting with >, a line saying
// who wrote what follows ("On Mon, Joe wrote:"), Outlook's "Original
// Message" or its From: block, and "-- " before a signature.

// Turn is whose move it is after a message: "theirs" when the person wrote
// it, "yours" when someone else did and asked them something, "" when it
// answers or closes what came before.
func Turn(fromMe bool, body string) string {
	if fromMe {
		return "theirs"
	}
	if Asks(body) {
		return "yours"
	}
	return ""
}

// Asks is whether a message asks its reader something: a question, or a
// please, in its own words.
func Asks(body string) bool {
	own := links.ReplaceAllString(OwnWords(body), "")
	lower := strings.ToLower(own)
	return strings.Contains(own, "?") || strings.Contains(lower, "please") || strings.Contains(lower, "let me know")
}

var (
	links = regexp.MustCompile(`(?i)\b(https?://|mailto:|www\.)\S+`)
	// wrote is the line a mail app puts above what it quotes.
	wrote = regexp.MustCompile(`(?i)^\s*(on .{4,200}wrote:|-+ ?original message ?-+|from: .+|-- ?)\s*$`)
)

// OwnWords is a message without what it quotes or its signature: the
// words its writer wrote this time.
func OwnWords(body string) string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if wrote.MatchString(line) {
			break
		}
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
