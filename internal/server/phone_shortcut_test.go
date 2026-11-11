package server_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// An iPhone shares to Sameway through a Shortcut: Save to Sameway makes
// it a link with a key of its own, shown once with the steps. The key
// lets a share in over the Wi-Fi and nothing else, not a page, not as a
// cookie; it is listed with the phones paired and can be removed.
func TestAnIPhoneSharesThroughAShortcut(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	h := srv.WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() {},
		LAN: func(bool) error { return nil }, LANBase: func() string { return "http://192.168.1.20:8080" }})
	postForm(t, h, "/phone/on", nil)
	if page := get(t, h, "/share").Body.String(); !strings.Contains(page, "Make a Shortcut for an iPhone") {
		t.Fatalf("Save to Sameway offers it: %s", truncate(page))
	}
	made := postForm(t, h, "/phone/shortcut", nil).Body.String()
	m := regexp.MustCompile(`value="(http://[^"]+/share\?key=([0-9a-f]+))"`).FindStringSubmatch(made)
	if m == nil || !strings.Contains(made, "Show in Share Sheet") || !strings.Contains(made, "<strong>files</strong>") {
		t.Fatalf("the link, with the steps: %s", truncate(made))
	}
	wifi := srv.LAN(h)
	send := func(method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		fw, _ := mw.CreateFormFile("files", "IMG_0042.png")
		fw.Write(pngOf(t, 8, 8))
		mw.Close()
		r := httptest.NewRequest(method, path, &body)
		r.Header.Set("Content-Type", mw.FormDataContentType())
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		wifi.ServeHTTP(w, r)
		return w
	}
	if res := send("POST", "/share?key="+m[2], nil); res.Code != http.StatusSeeOther || !strings.HasPrefix(res.Header().Get("Location"), "/t/note/") {
		t.Fatalf("a photo shared is saved: %d %s", res.Code, res.Body.String())
	}
	if res := send("GET", "/?key="+m[2], nil); res.Code != http.StatusForbidden {
		t.Errorf("the key opens nothing else: %d", res.Code)
	}
	if res := send("GET", "/", &http.Cookie{Name: "sameway_device", Value: m[2]}); res.Code != http.StatusForbidden {
		t.Errorf("nor as a cookie: %d", res.Code)
	}
	page := get(t, h, "/workspaces").Body.String()
	id := regexp.MustCompile(`iPhone Shortcut.*?name="id" value="([0-9a-f]+)"`).FindStringSubmatch(page)
	if id == nil {
		t.Fatalf("listed with the phones: %s", truncate(page))
	}
	postForm(t, h, "/phone/forget", map[string][]string{"id": {id[1]}})
	if res := send("POST", "/share?key="+m[2], nil); res.Code != http.StatusForbidden {
		t.Errorf("removed, it is out: %d", res.Code)
	}
}
