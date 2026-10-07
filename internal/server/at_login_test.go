package server_test

import (
	"os"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/atlogin"
)

// The owner has Sameway open at sign-in from Workspaces, and stops it
// there: the entry among what starts at sign-in is made and taken away.
func TestSamewayOpensAtSignInWhenAsked(t *testing.T) {
	home := t.TempDir()
	t.Setenv("APPDATA", home)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	was := atlogin.Shortcut
	atlogin.Shortcut = func(lnk, target, args, dir string, min bool) error {
		if !strings.Contains(args, "at-login --workspace") || !min {
			t.Errorf("the sign-in entry starts Sameway apart, minimised: %q %v", args, min)
		}
		return os.WriteFile(lnk, nil, 0o644)
	}
	defer func() { atlogin.Shortcut = was }()

	_, h := newApp(t)
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, "Open Sameway when I sign in") {
		t.Fatalf("Workspaces offers it: %s", truncate(page))
	}
	postForm(t, h, "/at-login", map[string][]string{"set": {"on"}})
	if !atlogin.On() || !strings.HasPrefix(atlogin.Path(), home) {
		t.Fatalf("the entry is made where this system looks: %s", atlogin.Path())
	}
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, "Stop opening Sameway when I sign in") {
		t.Errorf("and offers to stop: %s", truncate(page))
	}
	postForm(t, h, "/at-login", map[string][]string{"set": {"off"}})
	if atlogin.On() {
		t.Error("and stops")
	}
}
