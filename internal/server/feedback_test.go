package server_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Telling the makers shows everything that would go before anything goes:
// the person's words, the version and system, the kind of model, the last
// failures as the log says them, never a key or an address; and it opens
// the makers' page with it, where the person sends it, or not.
func TestTellingTheMakersShowsAllThatWouldGo(t *testing.T) {
	a, h := newApp(t)
	t.Setenv("SAMEWAY_TEST_KEY", "sk-ant-secret")
	records.Record(a.Store, "system", records.Change{Action: "failed", Detail: "Anthropic did not accept the key saved on this computer"})
	if help := get(t, h, "/help").Body.String(); !strings.Contains(help, "Something not right?") {
		t.Fatalf("Help offers it: %s", truncate(help))
	}
	page := postForm(t, h, "/feedback", url.Values{"what": {"The weekend plan went on the wrong days. It said Sunday 10 October."}}).Body.String()
	for _, want := range []string{`action="https://github.com/tristanlawrenceguy/sameway/issues/new"`, `method="get"`, "The weekend plan went on the wrong days", "Recent failures:", "did not accept the key", "Nothing has been sent"} {
		if !strings.Contains(page, want) {
			t.Errorf("the page has %s: %s", want, truncate(page))
		}
	}
	if strings.Contains(page, "sk-ant-") || strings.Contains(page, "127.0.0.1") || strings.Contains(page, a.Workspace.Dir) {
		t.Error("no key, address or folder goes")
	}
}
