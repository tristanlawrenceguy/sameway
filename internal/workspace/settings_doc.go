package workspace

import "strings"

// What a model is told of the settings, in set_setting's description,
// which the assistant and every MCP client read. Each is named with what
// it is, in its first words (how changes arrive; how large the words are),
// which is how a model finds the one a person means; the technical ones,
// that only someone setting up a model, a broker or a meeting service
// sets, by their key alone. The whole of what one is for comes back with
// a refusal (SettingDoc), for the one tried, not all of them every time.

// technical are the settings named rather than said whole.
var technical = []string{"server.", "llm.", "mcp.", "mqtt.", "meetings.", "actions.", "chat.", "notify.command", "backup.folder", "update.mode", "tailnet.peers"}

func isTechnical(key string) bool {
	for _, p := range technical {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

// SettingsDoc says each setting in a line, for a tool's description: the
// ones a person asks for by what they are, the technical ones by key, and
// what each takes.
func SettingsDoc() string {
	var b strings.Builder
	for i, s := range Settings {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(s.Key)
		if !isTechnical(s.Key) {
			b.WriteString(": " + firstPart(s.Doc))
		}
		if s.Kind == "enum" {
			b.WriteString(" (" + strings.Join(s.Values, ", ") + ")")
		}
	}
	return b.String()
}

// SettingDoc is what one setting is for, said whole, or "".
func SettingDoc(key string) string {
	for _, s := range Settings {
		if s.Key == key {
			return s.Doc
		}
	}
	return ""
}

// firstPart is what a setting is, the start of its doc: up to the first
// colon, semicolon or full stop outside brackets.
func firstPart(doc string) string {
	depth := 0
	for i := 0; i+1 < len(doc); i++ {
		switch doc[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ':', ';', '.':
			if depth == 0 && doc[i+1] == ' ' {
				return doc[:i]
			}
		}
	}
	return doc
}
