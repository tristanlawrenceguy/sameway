package tailnet_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/tailnet"
)

// A workspace that names no machine stays off the tailnet: nothing starts
// and nothing is said.
func TestNoNameMeansOff(t *testing.T) {
	said := []string{}
	err := tailnet.Start(context.Background(), tailnet.Config{Name: "  "}, http.NotFoundHandler(), nil,
		func(s tailnet.Status) { said = append(said, s.String()) })
	if err != nil {
		t.Fatal(err)
	}
	if len(said) != 0 {
		t.Errorf("an unnamed tailnet should say nothing, said %q", said)
	}
}

// Each step is said twice: a short line for the terminal, and what to do
// next with the link to do it, for the chat.
func TestEachStepSaysWhatToDoNext(t *testing.T) {
	cases := []struct {
		s          tailnet.Status
		line, word string
	}{
		{tailnet.Status{State: tailnet.SignIn, Link: "https://login.tailscale.com/a/x"}, "https://login.tailscale.com/a/x", "here: https://login.tailscale.com/a/x\n"},
		{tailnet.Status{State: tailnet.NeedsHTTPS, Link: tailnet.HTTPSSettings, Host: "home.t.ts.net"}, "https://home.t.ts.net/ needs HTTPS", "Enable HTTPS"},
		{tailnet.Status{State: tailnet.Ready, Link: "https://home.t.ts.net/"}, "https://home.t.ts.net/", "go to: https://home.t.ts.net/\n"},
		{tailnet.Status{State: tailnet.Off}, "off", "no longer on your Tailscale network"},
	}
	for _, c := range cases {
		if !strings.Contains(c.s.String(), c.line) {
			t.Errorf("%s: terminal line %q lacks %q", c.s.State, c.s.String(), c.line)
		}
		if !strings.Contains(c.s.Words(), c.word) {
			t.Errorf("%s: chat words %q lack %q", c.s.State, c.s.Words(), c.word)
		}
		// The chat is plain text that turns addresses into links; Markdown
		// would show as brackets.
		if strings.Contains(c.s.Words(), "](") {
			t.Errorf("%s: chat words carry Markdown: %q", c.s.State, c.s.Words())
		}
	}
}
