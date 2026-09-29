package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// One wording for a change, from a log entry or a reply's stored change.
func TestSay(t *testing.T) {
	st := newFullService(t).Store
	for _, c := range []struct {
		fields map[string]any
		linked bool
		want   chat.Words
	}{
		{map[string]any{"action": "set", "target": "ui.text", "detail": "large"}, false, chat.Words{"changed", "text size", "to Large"}},
		{map[string]any{"action": "set", "component": "ui.pace", "detail": "calm"}, false, chat.Words{"changed", "pace", "to Calm"}},
		{map[string]any{"action": "created", "component": "note", "detail": "Seeds"}, true, chat.Words{"created", "note", "Seeds"}},
		{map[string]any{"action": "removed", "target": "card", "undoes": "a1", "summary": "You undid: Assistant added card Plan"}, false, chat.Words{"undid", "", "Assistant added card Plan"}},
		{map[string]any{"action": "set", "undoes": "a2", "summary": "You undid: You set ui.text large"}, false, chat.Words{"undid", "", "You changed text size to Large"}},
	} {
		if got := chat.Say(st, c.fields); got != c.want {
			t.Errorf("Say(%v, %v) = %+v, want %+v", c.fields, c.linked, got, c.want)
		}
	}
}
