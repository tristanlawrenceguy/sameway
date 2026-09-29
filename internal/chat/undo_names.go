package chat

import (
	"regexp"
	"strings"
)

// alreadyHumanized matches an already-humanized setting change summary:
// "Assistant changed spacing to Wide" or "Assistant changed Room between
// lines and words to Wide". The first capture is the actor, second is the
// property name (label or description fragment), third is the value, fourth
// is any trailing text.
var alreadyHumanized = regexp.MustCompile(`^(.+?) changed (.+?) to ([A-Z].*)(.*)$`)

// anyFromFragment matches " from <CapitalLetter>..." anywhere in text,
// capturing the old value for output like "from Short".
var anyFromFragment = regexp.MustCompile(` from ([A-Z].*)`)

// descPrefixToKey maps the beginning of a setting's doc to its key, so raw
// descriptions stored in older summaries map back to setting keys for display.
var descPrefixToKey = map[string]string{
	"room between lines":                         "ui.spacing",
	"how large the words are":                    "ui.text",
	"how changes arrive":                         "ui.pace",
	"auto fades per-item controls until hovered": "ui.controls",
	"lists shown on pages":                       "ui.lists",
}

// cleanHumanized rewrites an already-humanized setting change summary back to
// the standard format, so description fragments and label names resolve to
// canonical key-based output. Em-dash descriptions (e.g., "Pace — how changes
// arrive") are stripped before parsing.
func cleanHumanized(s string) string {
	// Strip em-dash field descriptions from the summary before regex matching,
	// so "text size — the length of words from Short to Big" becomes
	// "text size from Short to Big". The whole " — <desc>" is removed and
	// replaced with a space.
	if i := strings.Index(s, " — "); i > 0 {
		s = strings.TrimSpace(s[:i]) + " " + strings.TrimSpace(s[i+5:])
	}

	m := alreadyHumanized.FindStringSubmatch(s)
	if m == nil {
		return s
	}
	actor := m[1]
	propFull := strings.ToLower(m[2])
	value := m[3]
	rest := m[4]

	key := lookupKey(propFull)
	if key != "" {
		name, val := settingWords(key, value)
		return actor + " changed " + name + " to " + val + rest
	}

	// Fallback: if prop starts with a known label followed by extra text (from
	// em-dash description remnants), try matching just the leading label.
	for label, key := range labelToKey {
		if !strings.HasPrefix(propFull, label+" ") && propFull != label {
			continue
		}
		name, val := settingWords(key, value)

		// Extract any "from X" fragment between the label and the value. Use the
		// original (non-lowercased) prop portion so capital letters in values are preserved.
		labelLen := len(label)
		if strings.HasPrefix(propFull, label+" ") {
			labelLen++ // skip trailing space
		}
		trailingOrig := m[2][labelLen:]

		// Match " from <CapitalLetter>..." anywhere in the remaining text.
		fromMatch := anyFromFragment.FindStringSubmatch(trailingOrig)
		if len(fromMatch) > 1 {
			return actor + " changed " + name + " from " + fromMatch[1] + " to " + val + rest
		}

		return actor + " changed " + name + " to " + val + rest
	}

	return s
}

// lookupKey maps a property name back to its setting key. It first checks
// known labels from settingNames (lowercased), then description prefixes.
func lookupKey(prop string) string {
	if key, ok := labelToKey[prop]; ok {
		return key
	}
	for prefix, key := range descPrefixToKey {
		if strings.HasPrefix(prop, prefix) {
			return key
		}
	}
	return ""
}

// labelToKey maps lowercased setting labels to their keys.
var labelToKey = map[string]string{
	"text size":       "ui.text",
	"spacing":         "ui.spacing",
	"pace":            "ui.pace",
	"item buttons":    "ui.controls",
	"lists shown":     "ui.lists",
	"developer pages": "ui.developer",
	"language":        "ui.language",
	"model provider":  "llm.provider",
	"model":           "llm.model",
}
