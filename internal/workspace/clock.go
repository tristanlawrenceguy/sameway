package workspace

import "github.com/tristanlawrenceguy/sameway/internal/when"

// Hours24 is whether this workspace's person reads times on the 24-hour
// clock: the one they chose (ui.clock), else their language's. It is read
// as the settings are now, so a change is said on the next page; and it is
// this workspace's alone, passed to what says a time rather than set on
// the program, so two workspaces open at once each keep theirs.
func (w *Workspace) Hours24() bool {
	return when.TwentyFour(w.Config.UI.Clock, w.Config.UI.Language)
}
