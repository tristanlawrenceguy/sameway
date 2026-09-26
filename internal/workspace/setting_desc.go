package workspace

import "strings"

// SettingDescription returns a human-readable short label for a setting key.
// For example "ui.pace" → ("pace", true), while "llm.model" → ("", false).
func SettingDescription(key string) (string, bool) {
	switch key {
	case "name":
		return "name", true
	case "server.addr":
		return "address", true
	case "ui.controls":
		return "controls", true
	case "ui.pace":
		return "pace", true
	case "ui.text":
		return "text size", true
	case "ui.spacing":
		return "spacing", true
	case "ui.needs":
		return "needs", true
	case "ui.language":
		return "language", true
	}
	return "", false
}

// SettingValueLabel returns the human-readable label for an enum value of a
// known setting. For ui.pace: calm→Calmly, quick→Quickly, still→All at once.
// Non-enum settings return the raw value unchanged (capitalized).
func SettingValueLabel(key, value string) string {
	switch key {
	case "ui.pace":
		switch value {
		case "calm":
			return "Calmly"
		case "quick":
			return "Quickly"
		case "still":
			return "All at once"
		}
	case "ui.controls":
		switch value {
		case "auto":
			return "Hover"
		case "visible":
			return "Always show"
		}
	case "ui.text":
		return strings.Title(value)
	case "ui.spacing":
		if value == "wide" {
			return "Wide"
		}
		return "Normal"
	}
	return strings.Title(value)
}

// SettingTarget checks whether a target string is a known setting key and,
// if so, returns its short label. Non-setting targets pass through unchanged.
func SettingTarget(target string) (string, bool) {
	label, ok := SettingDescription(target)
	if !ok {
		return target, false
	}
	return label, true
}
