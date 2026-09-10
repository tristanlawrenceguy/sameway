package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// ActivityType is the content type that logs every canvas change.
const ActivityType = "activity"

// Change is one canvas edit made during a turn or by a person. It is
// stored on assistant messages (as the receipt shown under the reply) and
// in the activity log (as the audit trail).
type Change struct {
	Action    string `json:"action"`
	Component string `json:"component,omitempty"`
	ID        string `json:"id,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

// Record writes one activity entry. Missing activity type is not an error:
// older workspaces simply have no log.
func Record(st *store.Store, actor string, c Change) {
	if _, ok := st.Types().Get(ActivityType); !ok {
		return
	}
	st.Create(ActivityType, map[string]any{
		"actor":     actor,
		"action":    c.Action,
		"target":    c.Component,
		"target_id": c.ID,
		"detail":    c.Detail,
	})
}

// Summarise turns a block's props into a short human label such as
// "Shopping" for a heading or "3 items" for a list, used in receipts.
func Summarise(component string, props map[string]any) string {
	pick := func(keys ...string) string {
		for _, k := range keys {
			if s, ok := props[k].(string); ok && s != "" {
				return truncate(s, 40)
			}
		}
		return ""
	}
	switch component {
	case "heading", "card", "text":
		return pick("text", "title", "content")
	case "table":
		return pick("caption")
	case "list":
		if items, ok := props["items"].([]any); ok {
			return fmt.Sprintf("%d items", len(items))
		}
	case "alert", "status":
		return pick("title", "message")
	case "button", "link", "badge", "text-field", "textarea", "select", "checkbox":
		return pick("label")
	}
	return ""
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}
