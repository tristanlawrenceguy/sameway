package cli_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/cli"
)

// Another site's page cannot post to the workspace (CSRF against
// localhost); Sameway's own pages, the bookmark's Save, a phone's share,
// and clients that are not browsers still can.
func TestOtherSitesCannotWrite(t *testing.T) {
	t.Parallel()
	dir := initWorkspace(t)
	a, err := app.Load(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	h := cli.Handler(a, "")
	start, _ := a.Store.Count("note")
	n := 0
	share := func(headers ...string) int {
		n++
		form := url.Values{"title": {"Kept"}, "text": {"words " + strings.Repeat("x", n)}}
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/share", strings.NewReader(form.Encode()))
		req.RemoteAddr = "127.0.0.1:5000"
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for i := 0; i+1 < len(headers); i += 2 {
			req.Header.Set(headers[i], headers[i+1])
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	for _, refused := range [][]string{
		{"Sec-Fetch-Site", "cross-site"},
		{"Sec-Fetch-Site", "same-site"}, // another port or name on this machine is another program
		{"Origin", "https://evil.example"},
	} {
		if code := share(refused...); code != http.StatusForbidden {
			t.Errorf("%v is refused: %d", refused, code)
		}
	}
	for _, passed := range [][]string{
		{"Sec-Fetch-Site", "same-origin", "Origin", "http://127.0.0.1:8080"}, // a form or fetch on Sameway's page
		{"Sec-Fetch-Site", "none"},          // a phone's share target, which the browser itself sends
		{"Origin", "http://127.0.0.1:8080"}, // an older browser, same origin
		{},                                  // the command line, MCP and API clients
	} {
		if code := share(passed...); code == http.StatusForbidden || code >= 500 {
			t.Errorf("%v passes: %d", passed, code)
		}
	}
	before, _ := a.Store.Count("note")
	if before != start+4 {
		t.Errorf("only the four that passed were saved: %d then %d", start, before)
	}
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/share?title=Kept&text=from+the+bookmark", nil)
	req.RemoteAddr = "127.0.0.1:5000"
	req.Header.Set("Sec-Fetch-Site", "cross-site") // the bookmark opens /share from the page being read
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "from the bookmark") {
		t.Errorf("the bookmark still opens the filled form: %d", rec.Code)
	}
	if after, _ := a.Store.Count("note"); after != before {
		t.Errorf("opening it saves nothing: %d then %d", before, after)
	}
}
