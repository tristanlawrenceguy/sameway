package workspace

import "strings"

// settingNames are what a person calls a setting, for every sentence that
// says one changed: the log, a reply's receipt, an undo. One list, so the
// same setting is never called two things. A setting not here is called by
// the last word of its key.
var settingNames = map[string]string{
	"ui.text":      "Text size",
	"ui.spacing":   "Spacing",
	"ui.pace":      "Pace",
	"ui.controls":  "Item buttons",
	"ui.lists":     "Lists shown",
	"ui.developer": "Developer pages",
	"ui.nest":      "Nested in the sidebar",
	"ui.language":  "Language",
	"ui.clock":     "Clock",
	"llm.provider": "Model provider",
	"llm.model":    "Model",
}

// SettingLabel is a setting's name in words: "ui.text" is "Text size".
func SettingLabel(key string) string {
	if name, ok := settingNames[key]; ok {
		return name
	}
	parts := strings.Split(key, ".")
	last := strings.ReplaceAll(parts[len(parts)-1], "_", " ")
	if last == "" {
		return key
	}
	return strings.ToUpper(last[:1]) + last[1:]
}
