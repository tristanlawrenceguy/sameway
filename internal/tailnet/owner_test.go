package tailnet

import (
	"testing"

	"tailscale.com/tailcfg"
)

// The person's own phone gets in; someone else's device, or a tagged
// server on the same tailnet, does not. A node signed in with a tagged
// key belongs to nobody, so the tailnet's own rules decide.
func TestOnlyTheOwnersDevicesGetIn(t *testing.T) {
	const me, them tailcfg.UserID = 1, 2
	mine := &tailcfg.Node{User: me}
	theirs := &tailcfg.Node{User: them}
	server := &tailcfg.Node{User: me, Tags: []string{"tag:server"}}
	if !admit(me, false, mine) {
		t.Error("the owner's own device should get in")
	}
	if admit(me, false, theirs) {
		t.Error("another person's device should not get in")
	}
	if admit(me, false, server) {
		t.Error("a tagged device is nobody's own and should not get in")
	}
	if !admit(me, true, theirs) {
		t.Error("a tagged node should leave who gets in to the tailnet's rules")
	}
}

// A device is named the way the tailnet names it, without the domain.
func TestADeviceIsCalledByItsTailnetName(t *testing.T) {
	cases := []struct {
		node *tailcfg.Node
		want string
	}{
		{&tailcfg.Node{ComputedName: "pixel-7", Name: "pixel-7.tail1234.ts.net."}, "pixel-7"},
		{&tailcfg.Node{Name: "iphone.tail1234.ts.net."}, "iphone"},
	}
	for _, c := range cases {
		if got := deviceName(c.node); got != c.want {
			t.Errorf("deviceName(%q) = %q, want %q", c.node.Name, got, c.want)
		}
	}
}
