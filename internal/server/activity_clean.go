package server

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// cleanActivityFields returns a copy of rec.Fields with values cleaned for
// human readability on set-type activity entries: "before" becomes just the
// value text (capitalised), "target" becomes the setting label, and "detail"
// is capitalised. Non-setting entries are returned unchanged.
func cleanActivityFields(rec *store.Record) map[string]any {
	action, _ := rec.Fields["action"].(string)
	target, _ := rec.Fields["target"].(string)
	if action != "set" || target == "" {
		return rec.Fields
	}
	if !strings.HasPrefix(target, "ui.") && !strings.HasPrefix(target, "llm.") {
		return rec.Fields
	}
	out := make(map[string]any, len(rec.Fields))
	for k, v := range rec.Fields {
		out[k] = v
	}
	cleanBefore(&out)
	cleanTarget(&out)
	cleanDetail(&out)
	return out
}

// cleanBefore extracts "value" from a {"value": ...} map and capitalises it.
func cleanBefore(fields *map[string]any) {
	if b, ok := (*fields)["before"].(map[string]any); ok {
		if v, has := b["value"]; has {
			(*fields)["before"] = capitalize(fmt.Sprint(v))
		}
	}
}

// cleanTarget converts ui.text → Text size via the workspace.
func cleanTarget(fields *map[string]any) {
	if t, ok := (*fields)["target"].(string); ok && (strings.HasPrefix(t, "ui.") || strings.HasPrefix(t, "llm.")) {
		(*fields)["target"] = workspace.SettingLabel(t)
	}
}

// cleanDetail capitalises the first letter of a detail value.
func cleanDetail(fields *map[string]any) {
	if d, ok := (*fields)["detail"].(string); ok && d != "" {
		(*fields)["detail"] = capitalize(d)
	}
}
