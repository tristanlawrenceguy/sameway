package records

import "strings"

// Whose turn it is in a conversation is worked out, not guessed: who wrote
// last, and whether they asked something. A small model asked the same
// missed half the questions left for the person ("Tuesday works. 3pm?");
// the question mark does not.

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
// please.
func Asks(body string) bool {
	lower := strings.ToLower(body)
	return strings.Contains(body, "?") || strings.Contains(lower, "please") || strings.Contains(lower, "let me know")
}
