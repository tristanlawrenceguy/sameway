package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A record's page on the phone opens it on the computer's screen; on the
// computer it offers a code that opens it on the phone. Each shows only
// where it can work, and nothing but a page of the workspace is opened.
func TestAPageMovesBetweenPhoneAndComputer(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	var shown []string
	h := server.New(a).WithFleet(&server.Fleet{Launch: func(string, string) error { return nil }, Exit: func() {},
		LAN: func(bool) error { return nil }, LANBase: func() string { return "http://192.168.1.20:8080" },
		Show: func(path string) error { shown = append(shown, path); return nil }})
	rec, _ := a.Store.Create("note", map[string]any{"title": "Garden plan"})
	page := "/t/note/" + rec.ID
	from := func(addr, method, path string, form url.Values) *httptest.ResponseRecorder {
		var r *http.Request
		if form != nil {
			r = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		} else {
			r = httptest.NewRequest(method, path, nil)
		}
		r.RemoteAddr = addr
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	const phone, computer = "100.101.102.103:5000", "127.0.0.1:5000"

	onPhone := from(phone, "GET", page, nil).Body.String()
	if !strings.Contains(onPhone, ">Open on the computer<") || strings.Contains(onPhone, ">On your phone<") {
		t.Fatalf("on the phone, Open on the computer: %s", truncate(onPhone))
	}
	if res := from(phone, "POST", "/computer/show", url.Values{"to": {page}}); res.Code != http.StatusSeeOther || len(shown) != 1 || shown[0] != page {
		t.Errorf("pressed, the computer opens the page: %d %v", res.Code, shown)
	}
	from(phone, "POST", "/computer/show", url.Values{"to": {"https://example.com/"}})
	if len(shown) != 1 {
		t.Errorf("nothing but a page of the workspace opens: %v", shown)
	}

	onComputer := from(computer, "GET", page, nil).Body.String()
	if !strings.Contains(onComputer, ">On your phone<") || strings.Contains(onComputer, ">Open on the computer<") {
		t.Fatalf("on the computer, On your phone: %s", truncate(onComputer))
	}
	code := from(computer, "GET", "/phone/here?to="+url.QueryEscape(page), nil).Body.String()
	if !strings.Contains(code, `aria-label="Code to scan with your phone"`) || !strings.Contains(code, "http://192.168.1.20:8080"+page) {
		t.Errorf("a code that opens it on the phone: %s", truncate(code))
	}
}
