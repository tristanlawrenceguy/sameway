package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, ">Use Sameway on your phone<") || !strings.Contains(page, "only on a Wi-Fi you trust") {
		t.Fatalf("Workspaces offers it, and says on which Wi-Fi: %s", truncate(page))
	}
	postForm(t, h, "/phone/on", nil)
	page := get(t, h, "/workspaces").Body.String()
	m := regexp.MustCompile(`open http://192\.168\.1\.20:8080 and type <strong>([B-Z]{4}-[B-Z]{4})</strong>`).FindStringSubmatch(page)
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

// Not everyone can scan: the code is also eight letters to type on the
// phone, read however they are typed; the page a phone not paired sees
// has a box for them; and after five wrong tries every code waiting is
// dropped, so the letters cannot be guessed.
func TestAPhoneCanTypeTheCodeInstead(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	h := srv.WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() {},
		LAN:     func(bool) error { return nil },
		LANBase: func() string { return "http://192.168.1.20:8080" }})
	codeOf := func() string {
		page := get(t, h, "/workspaces").Body.String()
		m := regexp.MustCompile(`open http://192\.168\.1\.20:8080 and type <strong>([B-Z]{4}-[B-Z]{4})</strong>`).FindStringSubmatch(page)
		if m == nil {
			t.Fatalf("the letters to type are shown beside the code: %s", truncate(page))
		}
		return m[1]
	}
	wifi := srv.LAN(h)
	ask := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		wifi.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	if res := ask("/").Body.String(); !strings.Contains(res, `<label for="code">`) || !strings.Contains(res, `action="/pair"`) {
		t.Errorf("a phone not paired can type the letters: %s", res)
	}
	typed := strings.ToLower(strings.Replace(codeOf(), "-", " ", 1))
	if res := ask("/pair?code=" + url.QueryEscape(typed)); res.Code != http.StatusSeeOther {
		t.Errorf("the letters typed small, with a space, pair the phone: %d", res.Code)
	}
	waiting := codeOf()
	for i := 0; i < 5; i++ {
		ask("/pair?code=BBBB-BBBB")
	}
	if res := ask("/pair?code=" + waiting); res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "mistyped") {
		t.Errorf("after five wrong tries the code waiting is dropped: %d", res.Code)
	}
}
