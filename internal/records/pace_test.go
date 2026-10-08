package records

import (
	"testing"
	"time"
)

// A key spends its changes and earns them back over the minute; another
// key's budget is its own.
func TestAKeysChangesComeBackOverTheMinute(t *testing.T) {
	now := time.Unix(0, 0)
	p := newPacer(3, time.Minute)
	p.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if wait := p.take("agent:a"); wait != 0 {
			t.Fatalf("change %d is within the budget, waited %v", i+1, wait)
		}
	}
	if wait := p.take("agent:a"); wait != 20*time.Second {
		t.Errorf("the fourth waits for one to come back, 20s, got %v", wait)
	}
	if wait := p.take("agent:b"); wait != 0 {
		t.Errorf("another key is not held back by this one: %v", wait)
	}
	now = now.Add(20 * time.Second)
	if wait := p.take("agent:a"); wait != 0 {
		t.Errorf("after 20s one change is back: %v", wait)
	}
	now = now.Add(time.Hour)
	for i := 0; i < 3; i++ {
		p.take("agent:a")
	}
	if wait := p.take("agent:a"); wait == 0 {
		t.Error("an hour away earns no more than the budget")
	}
}
