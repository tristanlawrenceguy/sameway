package speech

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// An engine that never started says what happened, not where it lives:
// Smart App Control blocking it is named, and the path is left out.
func TestAnEngineThatCannotStartSaysWhy(t *testing.T) {
	t.Parallel()
	blocked := stopped("the speech engine", errors.New(`fork/exec C:\Users\me\engine.exe: An Application Control policy has blocked this file.`), "")
	if !strings.Contains(blocked.Error(), "Smart App Control blocked it") || strings.Contains(blocked.Error(), `C:\`) || strings.Contains(blocked.Error(), "fork/exec") {
		t.Errorf("blocked: %v", blocked)
	}
	missing := stopped("the speech engine", exec.Command("sameway-no-such-engine").Run(), "")
	if missing.Error() != "the speech engine could not start on this computer" {
		t.Errorf("missing: %v", missing)
	}
}
