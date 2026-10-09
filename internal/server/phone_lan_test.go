package server_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A phone on the Wi-Fi gets in only by scanning the code on Workspaces: a
// link that works once leaves it a cookie that keeps it paired, as the
// owner; anything else on the Wi-Fi sees how to pair, and a phone taken
// away is out.
func TestAPhoneOnTheWiFiPairsByACode(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	on := false
	srv := server.New(a)
	h := srv.WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() {},
		LAN:     func(want bool) error { on = want; return nil },
		LANBase: func() string { return map[bool]string{true: "http://192.168.1.20:8080"}[on] }})
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, ">Use Sameway on your phone<") {
		t.Fatalf("Workspaces offers it: %s", truncate(page))
	}
	postForm(t, h, "/phone/on", nil)
	page := get(t, h, "/workspaces").Body.String()
	m := regexp.MustCompile(`http://192\.168\.1\.20:8080/pair\?code=([0-9a-f]+)`).FindStringSubmatch(page)
	if !on || m == nil || !strings.Contains(page, `aria-label="Code to scan with your phone"`) {
		t.Fatalf("on, with a code to scan: %s", truncate(page))
	}
	wifi := srv.LAN(h)
	ask := func(path string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("User-Agent", "Mozilla/5.0 (iPhone)")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		wifi.ServeHTTP(w, r)
		return w
	}
	if res := ask("/", nil); res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "scan the code") {
		t.Errorf("anything unpaired sees how to pair: %d", res.Code)
	}
	paired := ask("/pair?code="+m[1], nil)
	var cookie *http.Cookie
	for _, c := range paired.Result().Cookies() {
		cookie = c
	}
	if paired.Code != http.StatusSeeOther || cookie == nil || !cookie.HttpOnly {
		t.Fatalf("the code pairs the phone: %d %v", paired.Code, cookie)
	}
	if res := ask("/pair?code="+m[1], nil); res.Code != http.StatusForbidden {
		t.Error("a code works once")
	}
	if res := ask("/", cookie); res.Code != http.StatusOK {
		t.Errorf("a paired phone gets in: %d", res.Code)
	}
	page = get(t, h, "/workspaces").Body.String()
	id := regexp.MustCompile(`name="id" value="([0-9a-f]+)"`).FindStringSubmatch(page)
	if !strings.Contains(page, "iPhone") || id == nil {
		t.Fatalf("the phone is listed: %s", truncate(page))
	}
	postForm(t, h, "/phone/forget", map[string][]string{"id": {id[1]}})
	if res := ask("/", cookie); res.Code != http.StatusForbidden {
		t.Error("a phone taken away is out")
	}
}
