package server_test

import (
	"strings"
	"testing"
)

// The help page a newcomer opens does not lead with an administrator's
// setup in Microsoft's and Zoom's consoles: it is there, closed, under
// Transcripts from Teams or Zoom.
func TestHelpKeepsMeetingAppSetupClosed(t *testing.T) {
	_, h := newApp(t)
	page := get(t, h, "/help").Body.String()
	at := strings.Index(page, "Transcripts from Teams or Zoom")
	entra := strings.Index(page, "Microsoft Entra")
	if at < 0 || entra < at || !strings.Contains(page[strings.LastIndex(page[:at], "<details"):entra], "<details") {
		t.Errorf("the setup is inside a closed disclosure: %s", truncate(page))
	}
}
