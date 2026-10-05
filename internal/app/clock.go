package app

import (
	"github.com/tristanlawrenceguy/sameway/internal/when"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// sayTimes has every time of day said on the person's clock: the one they
// chose (ui.clock), else their language's. It is read live, so a change of
// setting is said on the next page (internal/when, design/foundations/glance.md).
func sayTimes(ws *workspace.Workspace) {
	when.Hours24 = func() bool { return when.TwentyFour(ws.Config.UI.Clock, ws.Config.UI.Language) }
}
