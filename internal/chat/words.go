package chat

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Words is a change as one line of activity says it: the verb, what it
// was done to, and more about it. The chat's Changes made list and the
// activity log both say a change through Say, so a wording is fixed once
// and reads the same in both.
type Words struct{ Action, Target, Detail string }

// Say is the words for a change, from its log entry's fields: action,
// target, detail, and for an undo, undoes and summary. A receipt's change
// is said the same way with its component as the target. linked says the
// line leads to the thing; an undo then keeps its colon, which the event
// adds itself where there is no link, so it is never said twice.
//
//	set ui.text large          changed text size to Large
//	created note Shopping      created note Shopping
//	(undo) You undid: ...      undid: Assistant added card Plan
func Say(f map[string]any, linked bool) Words {
	str := func(k string) string { s, _ := f[k].(string); return s }
	action, target, detail := str("action"), str("target"), str("detail")
	if target == "" {
		target = str("component")
	}
	if str("undoes") != "" {
		summary := CleanSummary(str("summary"))
		for _, verb := range []string{" undid: ", " put back: "} {
			if _, after, ok := strings.Cut(summary, verb); ok {
				verb = strings.TrimSpace(verb)
				if !linked {
					verb = strings.TrimSuffix(verb, ":")
				}
				return Words{Action: verb, Detail: after}
			}
		}
	}
	if isSettingChange(Change{Action: action, Component: target, Detail: detail}) {
		name, value := settingWords(target, detail)
		return Words{Action: "changed", Target: name, Detail: "to " + value}
	}
	return Words{Action: action, Target: target, Detail: detail}
}

// settingWords is a setting by its one name, from the workspace, and the
// value capitalised: text size, Large.
func settingWords(key, value string) (string, string) {
	return strings.ToLower(workspace.SettingLabel(key)), strings.ToUpper(value[:1]) + value[1:]
}
