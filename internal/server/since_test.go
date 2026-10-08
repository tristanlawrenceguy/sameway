package server_test

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// sinceSection is the notice on a page, or "".
func sinceSection(page string) string {
	i := strings.Index(page, `data-component="since"`)
	if i < 0 {
		return ""
	}
	rest := page[i:]
	return rest[:strings.Index(rest, "</section>")]
}

// Back after a long while, the notice says how many and since when, lists
// the newest five, and leads to the rest where they begin in the log.
func TestSinceSaysHowManyAndListsAFew(t *testing.T) {
	a, h := newApp(t)
	a.Store.SetMeta("last:owner", time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339Nano))
	for i := range 7 {
		records.Record(a.Store, "human", records.Change{Action: "added", Component: "note", Detail: fmt.Sprintf("Idea %d", i), By: "Hana", ByLogin: "hana@example.com"})
	}

	notice := sinceSection(get(t, h, "/").Body.String())
	if !strings.Contains(notice, "7 changes by others since <time") {
		t.Fatalf("the count and the time come first:\n%s", notice)
	}
	if !strings.Contains(notice, "Here until you press Got it.") {
		t.Error("it says how long it stays")
	}
	if n := strings.Count(notice, `data-component="event"`); n != 5 {
		t.Errorf("five changes are listed, not %d", n)
	}
	if strings.Contains(notice, `id="activity-`) {
		t.Error("the log keeps each change's anchor; the notice does not repeat it")
	}
	rest := regexp.MustCompile(`<a href="/activity#(activity-[^"]+)">2 more changes in Activity</a>`).FindStringSubmatch(notice)
	if rest == nil {
		t.Fatalf("the rest are counted in a link to the log:\n%s", notice)
	}
	log := get(t, h, "/activity").Body.String()
	if !strings.Contains(log, `id="`+rest[1]+`"`) {
		t.Errorf("the link leads to where the rest begin: %s", rest[1])
	}
	if strings.Count(log, `id="`+rest[1]+`"`) != 1 {
		t.Error("on the log itself, each anchor is there once")
	}
}

// Got it returns to the page it was pressed on, not Home.
func TestSinceGotItStaysOnThePage(t *testing.T) {
	a, h := newApp(t)
	a.Store.SetMeta("last:owner", time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339Nano))
	records.Record(a.Store, "human", records.Change{Action: "deleted", Component: "note", Detail: "Shopping", By: "Hana", ByLogin: "hana@example.com"})

	notice := sinceSection(get(t, h, "/activity").Body.String())
	if !strings.Contains(notice, `name="from" value="/activity"`) || !strings.Contains(notice, "1 change by others") {
		t.Fatalf("the notice knows its page and says one change:\n%s", notice)
	}
	res := postForm(t, h, "/since/seen", url.Values{"from": {"/activity"}})
	if to := res.Header().Get("Location"); to != "/activity" {
		t.Errorf("Got it returns to the log, not %q", to)
	}
	if res := postForm(t, h, "/since/seen", url.Values{"from": {"//evil.example"}}); res.Header().Get("Location") != "/" {
		t.Error("only a page of ours is returned to")
	}
}

// What others changed is for the people let in: a published page, read by
// anyone on the internet, never shows it, though nobody it knows was
// taken for the owner.
func TestAPublishedPageDoesNotSayWhatChanged(t *testing.T) {
	a, h := newApp(t)
	a.Store.SetMeta("last:owner", time.Now().Add(-2*time.Hour).UTC().Format(time.RFC3339Nano))
	records.Record(a.Store, "human", records.Change{Action: "added", Component: "note", Detail: "Private plans", By: "Hana", ByLogin: "hana@example.com"})
	if !strings.Contains(get(t, h, "/").Body.String(), "changes by others since") && !strings.Contains(get(t, h, "/").Body.String(), "change by others since") {
		t.Fatal("the owner is told, to start with")
	}
	n, _ := a.Store.Create("note", map[string]any{"title": "Sourdough"})
	a.Workspace.Config.Publish.Types = "note"
	page := public(t, h.(*server.Server).Public(nil), http.MethodGet, "/t/note/"+n.ID, "").Body.String()
	if !strings.Contains(page, "Sourdough") || strings.Contains(page, "by others since") || strings.Contains(page, "Private plans") {
		t.Errorf("the internet is not shown the log:\n%s", truncate(page))
	}
}
