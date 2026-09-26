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
	// Activity is the log entry this change was written to, so a receipt
	// can offer to undo it.
	Activity string `json:"activity,omitempty"`
	// Undoes is the entry this change reversed, when it is an undo.
	Undoes string `json:"undoes,omitempty"`
	// Via is the device the change was made from when it was not this
	// machine, such as a phone over the tailnet.
	Via string `json:"via,omitempty"`
	// By is the person who made it when that was not the owner, by name,
	// so the log says "Bob removed" rather than "You removed".
	By string `json:"by,omitempty"`
	// ByLogin is who made it by their Tailscale login, when known: the
	// owner's too, so a change reads as theirs by name on the other
	// computers that host the workspace.
	ByLogin string `json:"-"`
	// Before is the thing as it was before the change, kept in the log so
	// the change can be undone. It is not part of a receipt.
	Before map[string]any `json:"-"`
	// Undone is the sentence of the entry this change reversed, for the
	// sentence of this one; Redid says that entry was itself an undo, so
	// this one puts the original back rather than undoing it again.
	Undone string `json:"-"`
	Redid  bool   `json:"-"`
}

// Record writes one activity entry and returns its id. A missing activity
// type is not an error: older workspaces simply have no log, and a log
// without a before field is a log that cannot be undone. The summary is
// the whole event as one sentence, because that is what a list of events
// has to show.
func Record(st *store.Store, actor string, c Change) string {
	t, ok := st.Types().Get(ActivityType)
	if !ok {
		return ""
	}
	fields := map[string]any{
		"summary":   summarise(actor, c),
		"actor":     actor,
		"action":    c.Action,
		"target":    c.Component,
		"target_id": c.ID,
		"detail":    c.Detail,
		"undoes":    c.Undoes,
		"via":       c.Via,
		"by":        c.By,
		"by_login":  c.ByLogin,
	}
	if c.Before != nil {
		fields["before"] = c.Before
	}
	for k := range fields {
		if _, has := t.Field(k); !has {
			delete(fields, k)
		}
	}
	rec, err := st.Create(ActivityType, fields)
	if err != nil {
		return ""
	}
	return rec.ID
}

// summarise says what happened in a person's words: "Assistant added card
// Shopping", "You removed list Groceries", "System failed: no model".
func summarise(actor string, c Change) string {
	who := map[string]string{"human": "You", "assistant": "Assistant", "system": "System"}[actor]
	if who == "" {
		who = actor
	}
	if actor == "human" && c.By != "" {
		who = c.By
	}
	if c.Undone != "" {
		if c.Redid {
			return who + " put back: " + c.Undone
		}
		return who + " undid: " + c.Undone
	}
	parts := []string{who, c.Action}
	if c.Component != "" {
		parts = append(parts, c.Component)
	}
	if c.Detail != "" {
		parts = append(parts, c.Detail)
	}
	if strings.HasPrefix(c.Via, "through ") {
		return strings.Join(parts, " ") + ", " + c.Via
	}
	if c.Via != "" {
		return strings.Join(parts, " ") + ", on " + c.Via
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
		// Its name, so Remove To pack is not heard as removing some items.
		if l := pick("label"); l != "" {
			return l
		}
		if items, ok := props["items"].([]any); ok {
			return fmt.Sprintf("list of %d", len(items))
		}
	case "alert", "status":
		return pick("title", "message")
	case "button", "link", "badge", "text-field", "textarea", "select", "checkbox":
		return pick("label")
	case "record":
		recTitle := trimWords(pick("record"), 6)
		return strings.TrimSpace(pick("type") + " " + recTitle)
	case "calendar":
		if caption := pick("caption"); caption != "" {
			return caption
		}
		if month, err := time.Parse("2006-01", pick("month")); err == nil {
			return month.Format("January 2006")
		}
	case ComponentName:
		return "Conversation"
	case "search":
		if label := pick("label"); label != "" {
			return label
		}
		return "Search"
	}
	return ""
}

// trimWords cuts s to at most n words, appending an ellipsis when trimmed.
func trimWords(s string, n int) string {
	fields := strings.Fields(strings.TrimSpace(s))
	if len(fields) <= n {
		return s
	}
	return strings.Join(fields[:n], " ") + "…"
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(strings.SplitN(s, "\n", 2)[0])
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

// LocalEntry says which log entries stay on the computer that wrote them
// when others host the workspace too: what was said to the assistant, its
// questions and their answers, and changes to what stays local itself.
// Every other entry travels, so a change made elsewhere reads as whose it
// was and glows in their colour here.
func LocalEntry(local map[string]bool) func(typeName string, fields map[string]any) bool {
	private := map[string]bool{"said": true, "proposed": true, "agreed to": true, "declined": true, "cleared": true, "failed": true}
	return func(typeName string, fields map[string]any) bool {
		if typeName != ActivityType {
			return false
		}
		action, _ := fields["action"].(string)
		target, _ := fields["target"].(string)
		return private[action] || local[target] || target == "conversation"
	}
}
