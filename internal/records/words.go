package records

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// Words is a change as one line of activity says it: the verb, what it
// was done to, and more about it. The chat's Changes made list and the
// activity log both say a change through Say, so a wording is fixed once
// and reads the same in both.
type Words struct{ Action, Target, Detail string }

// Say is the words for a change, from its log entry's fields: action,
// target, detail, and for an undo, undoes and summary. A receipt's change
// is said the same way with its component as the target. An undo is its
// bare verb and the sentence of what it took back; the event puts the
// colon between them, so no verb carries punctuation into data-action.
//
//	set ui.text large          changed text size to Large
//	created note Shopping      created note Shopping
//	(undo) You undid: ...      undid, Assistant added card Plan
func Say(st *store.Store, f map[string]any) Words {
	str := func(k string) string { s, _ := f[k].(string); return s }
	action, target, detail := str("action"), str("target"), str("detail")
	if target == "" {
		target = str("component")
	}
	if str("undoes") != "" {
		summary := Sentence(st, f)
		for _, verb := range []string{" undid: ", " put back: "} {
			if _, after, ok := strings.Cut(summary, verb); ok {
				return Words{Action: strings.Trim(verb, " :"), Detail: after}
			}
		}
	}
	if isSettingChange(Change{Action: action, Component: target, Detail: detail}) {
		name, value := settingWords(target, detail)
		return Words{Action: "changed", Target: name, Detail: "to " + value}
	}
	return Words{Action: action, Target: personWord(target), Detail: strings.TrimLeft(detail, "-*• ")}
}

// personWord is what a person calls a thing the code calls otherwise: a
// canvas is a tab ("added canvas Weekend" was said). A detail that began
// as a list's first line keeps its words, not its dash.
func personWord(target string) string {
	if target == "canvas" {
		return "tab"
	}
	return target
}

// settingWords is a setting by its one name, from the workspace, and the
// value capitalised: text size, Large.
func settingWords(key, value string) (string, string) {
	return strings.ToLower(workspace.SettingLabel(key)), strings.ToUpper(value[:1]) + value[1:]
}
