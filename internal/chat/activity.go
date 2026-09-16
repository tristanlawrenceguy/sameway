package chat

import (
	"fmt"
	"strings"
	"time"

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
	// Href is the page of the thing changed, so a receipt and a log entry
	// lead to it: a block's own page, a record's page, a tab. Empty when
	// the thing is gone.
	Href string `json:"href,omitempty"`
}

// Record writes one activity entry. Missing activity type is not an error:
// older workspaces simply have no log. The summary is the whole event as
// one sentence, because that is what a list of events has to show.
func Record(st *store.Store, actor string, c Change) {
	if _, ok := st.Types().Get(ActivityType); !ok {
		return
	}
	st.Create(ActivityType, map[string]any{
		"summary":   summarise(actor, c),
		"actor":     actor,
		"action":    c.Action,
		"target":    c.Component,
		"target_id": c.ID,
		"detail":    c.Detail,
	})
}

// summarise says what happened in a person's words: "Assistant added card
// Shopping", "You removed list Groceries", "System failed: no model".
func summarise(actor string, c Change) string {
	who := map[string]string{"human": "You", "assistant": "Assistant", "system": "System"}[actor]
	if who == "" {
		who = actor
	}
	parts := []string{who, c.Action}
	if c.Component != "" {
		parts = append(parts, c.Component)
	}
	if c.Detail != "" {
		parts = append(parts, c.Detail)
	}
	return strings.Join(parts, " ")
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
	case "record":
		return strings.TrimSpace(pick("type") + " " + pick("record"))
	case "calendar":
		if caption := pick("caption"); caption != "" {
			return caption
		}
		if month, err := time.Parse("2006-01", pick("month")); err == nil {
			return month.Format("January 2006")
		}
	case ComponentName:
		return "Conversation"
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
