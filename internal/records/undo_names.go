package records

import (
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// alreadyHumanized matches an already-humanized setting change summary:
// "Assistant changed spacing to Wide" or "Assistant changed Room between
// lines and words to Wide". The first capture is the actor, second is the
// property name (label or description fragment), third is the value, fourth
// is any trailing text.
var alreadyHumanized = regexp.MustCompile(`^(.+?) changed (.+?) to ([A-Z].*)(.*)$`)

// cleanHumanized rewrites an already-humanized setting change summary back to
// the standard format, so description fragments and label names resolve to
// canonical key-based output.
func cleanHumanized(s string) string {
	m := alreadyHumanized.FindStringSubmatch(s)
	if m == nil {
		return s
	}
	actor := m[1]
	prop := strings.ToLower(m[2])
	value := m[3]
	rest := m[4]

	key := lookupKey(prop)
	if key != "" {
		name, val := settingWords(key, value)
		return actor + " changed " + name + " to " + val + rest
	}
	return s
}

// lookupKey maps what an older summary called a setting back to its key:
// its label ("text size"), or the start of what the setting says it is
// ("how changes arrive", "pace — how changes arrive"). Both come from the
// settings themselves (workspace.Settings, workspace.SettingLabel), so a
// setting renamed or added is found without a list here to keep in step.
func lookupKey(prop string) string {
	prop = strings.ToLower(strings.TrimSpace(prop))
	for _, st := range workspace.Settings {
		if strings.ToLower(workspace.SettingLabel(st.Key)) == prop {
			return st.Key
		}
	}
	for _, st := range workspace.Settings {
		doc := strings.ToLower(st.Doc)
		if i := strings.IndexAny(doc, ":;,"); i > 0 {
			doc = doc[:i]
		}
		if doc != "" && (strings.HasPrefix(prop, doc) || strings.Contains(prop, "— "+doc) || strings.HasPrefix(doc, prop)) {
			return st.Key
		}
	}
	return ""
}
