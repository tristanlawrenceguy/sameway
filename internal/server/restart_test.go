package server_test

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
	"github.com/tristanlawrenceguy/sameway/internal/update"
)

// A version installed and waiting is offered to the owner with Restart
// Sameway, which starts it and comes back by itself.
func TestAnInstalledUpdateOffersARestart(t *testing.T) {
	was := update.Pending
	update.Pending = func() string { return "0.2.0" }
	defer func() { update.Pending = was }()
	a, _ := newApp(t)
	restarted := 0
	h := server.New(a).WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() {}, Restart: func() error { restarted++; return nil }})
	page := get(t, h, "/").Body.String()
	if !strings.Contains(page, "Sameway 0.2.0 is installed") || !strings.Contains(page, ">Restart Sameway<") {
		t.Fatalf("the owner is offered the restart: %s", truncate(page))
	}
	res := postForm(t, h, "/restart", nil)
	if restarted != 1 || !strings.Contains(res.Body.String(), "Sameway is restarting") || !strings.Contains(res.Header().Get("Refresh"), "url=/") {
		t.Errorf("restarting starts it and the page comes back (%d): %s", restarted, truncate(res.Body.String()))
	}
}
