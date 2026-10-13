package workspace_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/examples"
	"github.com/tristanlawrenceguy/sameway/internal/workspace"
)

// A workspace's time zone is a place's name from the time zone database,
// built in so a computer without one knows it; a name that is not one is
// refused with how to give it, and empty is this computer's zone.
func TestTheZoneIsAPlacesName(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := workspace.Init(dir, examples.FS, examples.StarterRoot, false); err != nil {
		t.Fatal(err)
	}
	w, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if w.Zone() != time.Local {
		t.Errorf("unset, the zone is this computer's, got %v", w.Zone())
	}
	if err := w.Set("ui.zone", "Madrid"); err == nil || !strings.Contains(err.Error(), "such as Europe/Madrid") {
		t.Errorf("a name that is not a zone is refused with how to give it, got %v", err)
	}
	if err := w.Set("ui.zone", "America/Argentina/Buenos_Aires"); err != nil {
		t.Fatal(err)
	}
	if got := w.Zone().String(); got != "America/Argentina/Buenos_Aires" {
		t.Errorf("the zone set is the zone, got %s", got)
	}
	if err := w.Set("ui.zone", ""); err != nil || w.Zone() != time.Local {
		t.Errorf("emptied, it is this computer's again: %v %v", err, w.Zone())
	}
}
