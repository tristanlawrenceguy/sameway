package workspace

import "strings"

// What the assistant is told of the settings, every turn. The ones a
// person asks for in their own words (slower, larger text, the sidebar,
// publishing a tab) are said whole, since their words are how the
// assistant knows which to change. The ones only someone setting up a
// model, a broker or a meeting service sets are named with what they
// take; what one is for comes back with a refusal (SettingDoc), for the
// one it tries, not all of them every turn.

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
// ones a person asks for whole, the technical ones by key and what they
// take.
func SettingsDoc() string {
	var b strings.Builder
	for i, s := range Settings {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(s.Key)
		if !isTechnical(s.Key) {
			b.WriteString(": " + s.Doc)
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
