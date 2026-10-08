package records

import (
	"strconv"
	"sync"
	"time"
)

// An agent with a key of its own may change the workspace at a person's
// pace and a little more, never at a loop's: one told to tidy up and
// going wrong would otherwise rewrite every record before anyone looked.
// Each key has a budget of changes that refills over a minute; reading
// is never paced, and nor is the owner at this computer, who has no key.

// PaceChanges is how many changes an agent's key may make in a minute.
const PaceChanges = 60

type pacer struct {
	mu    sync.Mutex
	per   time.Duration // to earn one change back
	most  float64
	now   func() time.Time
	left  map[string]float64
	since map[string]time.Time
}

func newPacer(changes int, over time.Duration) *pacer {
	return &pacer{per: over / time.Duration(changes), most: float64(changes), now: time.Now,
		left: map[string]float64{}, since: map[string]time.Time{}}
}

var agentPace = newPacer(PaceChanges, time.Minute)

// Pace spends one of a key's changes, named by its login (agent:<id>),
// and says how long to wait first when none are left: 0 to go ahead.
func Pace(login string) time.Duration { return agentPace.take(login) }

func (p *pacer) take(login string) time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := p.now()
	left, seen := p.left[login]
	if !seen {
		left = p.most
	} else {
		left += float64(now.Sub(p.since[login])) / float64(p.per)
		if left > p.most {
			left = p.most
		}
	}
	p.since[login] = now
	if left < 1 {
		p.left[login] = left
		return time.Duration((1 - left) * float64(p.per))
	}
	p.left[login] = left - 1
	return 0
}

// SlowDown says what an agent past its pace does next.
func SlowDown(wait time.Duration) string {
	secs := int(wait/time.Second) + 1
	return "this key has made " + strconv.Itoa(PaceChanges) + " changes in the last minute, as many as one may; wait " + strconv.Itoa(secs) +
		"s and send it again. Reading is not limited, and the owner can make the change at the workspace"
}
