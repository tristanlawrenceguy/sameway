package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// Sameway runs in the background, so the owner stops it from the page:
// Quit Sameway on Workspaces says it has stopped and how to open it again,
// then stops.
func TestQuitSamewayFromThePage(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	stopped := 0
	h := server.New(a).WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() { stopped++ }})
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, ">Quit Sameway<") || !strings.Contains(page, `action="/quit"`) {
		t.Errorf("Workspaces offers Quit Sameway: %s", truncate(page))
	}
	page := postForm(t, h, "/quit", nil).Body.String()
	if !strings.Contains(page, "Sameway has stopped") || !strings.Contains(page, "Open it again") || stopped != 1 {
		t.Errorf("quitting says so and stops (%d): %s", stopped, truncate(page))
	}
}
