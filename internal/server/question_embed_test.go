package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// The box to ask in goes on the owner's own website: they copy a snippet
// from their ask page, and it shows the box alone, no header and no
// footer, its answers staying inside their site. Only the owner is given
// the snippet.
func TestTheAskBoxGoesOnTheOwnersWebsite(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	internet := h.(*server.Server).Public(nil)
	a.Workspace.Config.Publish.Types = "note"
	a.Chat.Address = func() string { return "https://tris.example.ts.net/" }

	mine := get(t, h, "/question").Body.String()
	if !strings.Contains(mine, `&lt;iframe src=&#34;https://tris.example.ts.net/question?embed&#34;`) {
		t.Fatalf("the owner is given the snippet, at the address the internet reaches: %s", truncate(mine))
	}
	if page := public(t, internet, http.MethodGet, "/question", "").Body.String(); strings.Contains(page, "iframe") || !strings.Contains(page, `class="sw-header"`) {
		t.Errorf("a visitor is not, and the page is a page: %s", truncate(page))
	}
	if rec := get(t, h, "/"); rec.Header().Get("X-Frame-Options") != "DENY" || rec.Header().Get("Content-Security-Policy") != "frame-ancestors 'none'" {
		t.Errorf("no other page goes inside another site's: %v", rec.Header())
	}
	shown := public(t, internet, http.MethodGet, "/question?embed", "")
	if shown.Header().Get("X-Frame-Options") != "" {
		t.Errorf("the box does: %v", shown.Header())
	}
	embedded := shown.Body.String()
	if strings.Contains(embedded, `class="sw-header"`) || strings.Contains(embedded, `class="sw-footer"`) || !strings.Contains(embedded, `action="/question?embed"`) {
		t.Fatalf("embedded, it is the box alone: %s", truncate(embedded))
	}
	a.Chat.Provider = &scripted{steps: []*llm.Response{{Text: "Weekdays 8 to 6."}}}
	req := httptest.NewRequest(http.MethodPost, "/question?embed", strings.NewReader(url.Values{"question": {"When are you open?"}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	internet.ServeHTTP(rec, req)
	if body := rec.Body.String(); !strings.Contains(body, "Weekdays 8 to 6") || strings.Contains(body, `class="sw-header"`) {
		t.Errorf("and its answer stays inside: %s", truncate(body))
	}

	// Before the internet reaches it, the owner on localhost is given no
	// snippet that would work nowhere else.
	a.Chat.Address = func() string { return "" }
	local := httptest.NewRequest(http.MethodGet, "/question", nil)
	local.Host = "127.0.0.1:8080"
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, local)
	if body := rec.Body.String(); strings.Contains(body, "iframe") || !strings.Contains(body, "Once the internet reaches this workspace") {
		t.Errorf("no snippet for localhost: %s", truncate(body))
	}
}

// Someone who can use only the box, in another site's page, by keyboard
// or screen reader: no skip link to a header that is not there, what
// they asked said back with the answer and a skip link to it, who they
// are kept for the next question, and an empty question is the field's
// own error.
func TestTheEmbeddedAskBoxWorksForEveryone(t *testing.T) {
	t.Parallel()
	a, h := newAppAt(t, tonight)
	internet := h.(*server.Server).Public(nil)
	a.Workspace.Config.Publish.Types = "note"
	ask := func(form url.Values) string {
		req := httptest.NewRequest(http.MethodPost, "/question?embed", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		internet.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	if page := public(t, internet, http.MethodGet, "/question?embed", "").Body.String(); strings.Contains(page, "Skip to main content") || !strings.Contains(page, `required aria-required="true"`) {
		t.Errorf("embedded, the first stop is the box, which must be filled: %s", truncate(page))
	}
	a.Chat.Provider = &scripted{steps: []*llm.Response{{Text: "Weekdays 8 to 6."}}}
	body := ask(url.Values{"question": {"When are you open?"}, "who": {"Sam, sam@example.com"}})
	for _, want := range []string{`href="#answer">Skip to the answer`, `id="answer"`, "You asked: <span dir=\"auto\">When are you open?", `value="Sam, sam@example.com"`} {
		if !strings.Contains(body, want) {
			t.Errorf("an answer says what was asked and keeps who asked (%s): %s", want, truncate(body))
		}
	}
	body = ask(url.Values{"question": {"  "}})
	if !strings.Contains(body, `aria-invalid="true"`) || !strings.Contains(body, "Say what you want to ask.") || !strings.Contains(body, `href="#question">Skip to what to fix`) {
		t.Errorf("an empty question is the field's error: %s", truncate(body))
	}
}
