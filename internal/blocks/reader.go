package blocks

import "time"

// A block is worked out for the person reading it: the time it is for
// them (Workspace.Clock) and their clock face, 12 or 24 hours (ui.clock,
// else their language's). Both are the workspace's own, never the
// program's, so two workspaces open at once each say times their way.

// now is the time it is for the person.
func (w *Workspace) now() time.Time {
	if w.Clock == nil {
		return time.Now()
	}
	return w.Clock()
}

// H24 is whether the person reads times on the 24-hour clock.
func (w *Workspace) H24() bool { return w.Settings != nil && w.Settings.Hours24() }
