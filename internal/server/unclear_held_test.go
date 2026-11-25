package server_test

import (
	"strings"
	"testing"
)

// A part of a photo read unsure is never sent on its own: an action set
// off by a record with [unclear] in what it would send is held and put to
// the owner, naming the part; Run it refuses while it is still unclear,
// and once put right it runs with the record as it is then.
func TestAnUnclearAmountIsHeldUntilChecked(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	a.Automate.StartWorker()
	h, url := listen(t)
	a.Store.Create("action", map[string]any{"title": "Pay bills", "kind": "webhook", "url": url + "/pay", "method": "POST",
		"body": "Pay {{title}}", "when": "added", "what": "task"})
	acceptAll(t, a)
	bill := written(t, a.Store, "created", "task", "", map[string]any{"title": "Water bill £[unclear: 18 or 78]"})
	h.quiet(t)
	var asked []string
	for i := 0; i < 50 && len(asked) == 0; i++ {
		for _, p := range a.Chat.Proposals() {
			d, _ := p.Fields["detail"].(string)
			asked = append(asked, d)
		}
		if len(asked) == 0 {
			h.quiet(t)
		}
	}
	if len(asked) != 1 || !strings.Contains(asked[0], "£[unclear: 18 or 78]") || !strings.Contains(asked[0], "Nothing was sent") {
		t.Fatalf("the owner is asked once, with the unclear part named: %q", asked)
	}
	held := a.Chat.Proposals()[0].ID
	if err := a.Chat.Accept(held); err == nil || !strings.Contains(err.Error(), "still has a part not read for sure") {
		t.Errorf("Run it refuses while the part is still unclear: %v", err)
	}
	h.quiet(t)

	written(t, a.Store, "updated", "task", bill.ID, map[string]any{"title": "Water bill £78"})
	for _, p := range a.Chat.Proposals() {
		a.Chat.Accept(p.ID)
	}
	if got := h.wait(t, 1); !strings.Contains(got[0], "Pay Water bill £78") {
		t.Errorf("Run it sends the record as it is now: %q", got)
	}
}
