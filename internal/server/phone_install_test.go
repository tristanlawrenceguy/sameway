package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A phone paired on the Wi-Fi is shown how to put Sameway on its home
// screen, in its own words. The manifest and icons reach it there (asked
// for with the cookie, or by anyone: they hold nothing of the workspace),
// and the manifest a paired phone gets starts the app at a link that
// pairs the home-screen app once, since an iPhone keeps that app's
// cookies apart from Safari's.
func TestAPhonePutsSamewayOnItsHomeScreen(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	h := srv.WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() {},
		LAN: func(bool) error { return nil }, LANBase: func() string { return "http://192.168.1.20:8080" }})
	postForm(t, h, "/phone/on", nil)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `<link rel="manifest" href="/manifest.webmanifest" crossorigin="use-credentials">`) {
		t.Errorf("the manifest is asked for with the phone's cookie")
	}
	code := regexp.MustCompile(`<strong>([B-Z]{4}-[B-Z]{4})</strong>`).FindStringSubmatch(get(t, h, "/workspaces").Body.String())
	if code == nil {
		t.Fatal("no code")
	}
	wifi := srv.LAN(h)
	ask := func(path string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) Safari")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		wifi.ServeHTTP(w, r)
		return w
	}
	cookieOf := func(res *httptest.ResponseRecorder) *http.Cookie {
		for _, c := range res.Result().Cookies() {
			return c
		}
		return nil
	}
	start := func(res *httptest.ResponseRecorder) string {
		var m struct {
			Start string `json:"start_url"`
		}
		json.Unmarshal(res.Body.Bytes(), &m)
		return m.Start
	}

	if res := ask("/manifest.webmanifest", nil); res.Code != 200 || start(res) != "/" {
		t.Errorf("the manifest reaches anything on the Wi-Fi, plain: %d %q", res.Code, start(res))
	}
	if res := ask("/icon-square-180.png", nil); res.Code != 200 || res.Header().Get("Content-Type") != "image/png" {
		t.Errorf("so do the icons: %d", res.Code)
	}
	if res := ask("/t/note", nil); res.Code != http.StatusForbidden {
		t.Errorf("and nothing of the workspace: %d", res.Code)
	}

	paired := ask("/pair?code="+code[1], nil)
	safari := cookieOf(paired)
	if paired.Header().Get("Location") != "/install" {
		t.Fatalf("a phone just paired is shown how to put it on its home screen: %s", paired.Header().Get("Location"))
	}
	if page := ask("/install", safari).Body.String(); !strings.Contains(page, "Add to Home Screen") || !strings.Contains(page, "Not now") {
		t.Errorf("in an iPhone's words: %s", truncate(page))
	}
	mf := ask("/manifest.webmanifest", safari)
	link := start(mf)
	if !strings.HasPrefix(link, "/pair?app=") {
		t.Fatalf("a paired phone's manifest starts at a link that pairs its app: %q", link)
	}
	if !strings.Contains(mf.Body.String(), `"url":"/computer/send"`) || !strings.Contains(mf.Body.String(), `"url":"/scan"`) {
		t.Errorf("the owner's phone holds Send to the computer and Scan to the list on its icon: %s", mf.Body.String())
	}
	if again := start(ask("/manifest.webmanifest", safari)); again != link {
		t.Errorf("the same link while it is unused: %q", again)
	}
	if res := ask(link, safari); res.Code != http.StatusSeeOther || cookieOf(res) != nil {
		t.Errorf("Safari, paired already, just goes in: %d", res.Code)
	}
	opened := ask(link, nil)
	app := cookieOf(opened)
	if opened.Code != http.StatusSeeOther || app == nil {
		t.Fatalf("the home-screen app, without Safari's cookie, is paired by it: %d", opened.Code)
	}
	if res := ask("/", app); res.Code != 200 {
		t.Errorf("and gets in: %d", res.Code)
	}
	if res := ask(link, nil); res.Code != http.StatusForbidden {
		t.Errorf("the link pairs once: %d", res.Code)
	}
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, "iPhone app") {
		t.Errorf("the app is listed with the phones, to be removed like one")
	}
}
