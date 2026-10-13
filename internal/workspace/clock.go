package workspace

import (
	"sync"
	"time"
	// The time zone database, built in: a computer without one (Windows)
	// would know no zone by name.
	_ "time/tzdata"

	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// Hours24 is whether this workspace's person reads times on the 24-hour
// clock: the one they chose (ui.clock), else their language's. It is read
// as the settings are now, so a change is said on the next page; and it is
// this workspace's alone, passed to what says a time rather than set on
// the program, so two workspaces open at once each keep theirs.
func (w *Workspace) Hours24() bool {
	return when.TwentyFour(w.Config.UI.Clock, w.Config.UI.Language)
}

// Zone is the time zone this workspace's person plans their days in: the
// one chosen (ui.zone), else this computer's. A workspace hosted on a
// server elsewhere, or on a laptop that travels, keeps its person's days
// rather than the computer's. It is read as the settings are now, so a
// change applies to the next thing worked out, with no restart.
func (w *Workspace) Zone() *time.Location {
	if loc, ok := zone(w.Config.UI.Zone); ok {
		return loc
	}
	return time.Local
}

// KnownZone is whether name is a time zone this program knows: a place's
// name from the time zone database, UTC, or Local for this computer's.
func KnownZone(name string) bool {
	_, ok := zone(name)
	return ok
}

// zones are the time zones read so far, by name: reading one from the
// database each time a page asks the time would be slow.
var zones sync.Map

func zone(name string) (*time.Location, bool) {
	if name == "" {
		return nil, false
	}
	if loc, ok := zones.Load(name); ok {
		return loc.(*time.Location), true
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, false
	}
	zones.Store(name, loc)
	return loc, true
}
