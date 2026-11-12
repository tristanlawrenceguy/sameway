package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// From the phone, a link opens in the computer's browser and words go on
// its clipboard; a link's trackers are dropped on the way, and nothing but
// a web address is opened.
func TestThePhoneSendsToTheComputer(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	var shown, copied []string
	h := server.New(a).WithFleet(&server.Fleet{Launch: func(string, string) error { return nil }, Exit: func() {},
		Show: func(to string) error { shown = append(shown, to); return nil },
		Copy: func(text string) error { copied = append(copied, text); return nil }})
	send := func(what string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/computer/send", strings.NewReader(url.Values{"what": {what}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.RemoteAddr = "100.101.102.103:5000"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}

	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest("GET", "/computer/send", nil))
	if body := page.Body.String(); !strings.Contains(body, ">Send to the computer<") {
		t.Fatalf("the page to send from: %s", truncate(body))
	}
	if res := send("  https://example.com/recipe?utm_source=x&id=4 "); res.Code != http.StatusSeeOther || len(shown) != 1 || shown[0] != "https://example.com/recipe?id=4" {
		t.Errorf("a link opens on the computer, without its trackers: %d %v", res.Code, shown)
	}
	if send("14 Garden Lane\nLeeds"); len(copied) != 1 || copied[0] != "14 Garden Lane\nLeeds" {
		t.Errorf("words go on the clipboard: %v", copied)
	}
	send("javascript://alert(1)")
	if len(shown) != 1 {
		t.Errorf("nothing but a web address opens: %v", shown)
	}
}
