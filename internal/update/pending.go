package update

import "sync"

// An installed version runs from the next start, and Sameway in the
// background, opened at sign-in, may not be started again for weeks. The
// version installed and waiting is kept here, so the page can offer to
// restart into it.
var pending struct {
	sync.Mutex
	version string
}

// Pending is the version installed and waiting for a restart, or "". A
// variable, so a page can be shown with one waiting.
var Pending = waiting

func waiting() string {
	pending.Lock()
	defer pending.Unlock()
	return pending.version
}

func setPending(version string) {
	pending.Lock()
	pending.version = version
	pending.Unlock()
}
