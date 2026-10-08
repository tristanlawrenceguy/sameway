package chat_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// One wording for a change, from a log entry or a reply's stored change.
func TestSay(t *testing.T) {
	st := newFullService(t).Store
	for _, c := range []struct {
		fields map[string]any
		linked bool
		want   records.Words
	}{
		{map[string]any{"action": "set", "target": "ui.text", "detail": "large"}, false, records.Words{Action: "changed", Target: "text size", Detail: "to Large"}},
		{map[string]any{"action": "set", "component": "ui.pace", "detail": "calm"}, false, records.Words{Action: "changed", Target: "pace", Detail: "to Calm"}},
		{map[string]any{"action": "created", "component": "note", "detail": "Seeds"}, true, records.Words{Action: "created", Target: "note", Detail: "Seeds"}},
		{map[string]any{"action": "removed", "target": "card", "undoes": "a1", "summary": "You undid: Assistant added card Plan"}, false, records.Words{Action: "undid", Target: "", Detail: "Assistant added card Plan"}},
		{map[string]any{"action": "set", "undoes": "a2", "summary": "You undid: You set ui.text large"}, false, records.Words{Action: "undid", Target: "", Detail: "You changed text size to Large"}},
	} {
		if got := records.Say(st, c.fields); got != c.want {
			t.Errorf("Say(%v, %v) = %+v, want %+v", c.fields, c.linked, got, c.want)
		}
	}
}
