package server_test

import (
	"encoding/json"
	"strings"
	"testing"
)

// On a phone's home screen Sameway is named Sameway, not after the page it
// was added from; held, its icon offers Save, Chat, Today and Search; and
// opened while the computer is out of reach it says so, with Try again,
// keeping no page of the workspace.
func TestTheHomeScreenAppIsNamedHeldAndSaysWhenOutOfReach(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `<meta name="apple-mobile-web-app-title" content="Sameway">`) {
		t.Errorf("an iPhone names it Sameway")
	}
	var m struct {
		ID        string `json:"id"`
		Shortcuts []struct {
			Name, URL string
		} `json:"shortcuts"`
	}
	if err := json.Unmarshal(get(t, h, "/manifest.webmanifest").Body.Bytes(), &m); err != nil || m.ID != "/" {
		t.Fatalf("the manifest: %v %+v", err, m)
	}
	var urls []string
	for _, s := range m.Shortcuts {
		urls = append(urls, s.URL)
	}
	if got := strings.Join(urls, " "); got != "/share /chat#message /today /search" {
		t.Errorf("held, the icon offers: %s", got)
	}
	for _, u := range urls {
		if res := get(t, h, strings.Split(u, "#")[0]); res.Code != 200 {
			t.Errorf("%s opens: %d", u, res.Code)
		}
	}
	sw := get(t, h, "/sw.js")
	body := sw.Body.String()
	if sw.Code != 200 || !strings.HasPrefix(sw.Header().Get("Content-Type"), "text/javascript") || !strings.Contains(body, "cannot be reached") || !strings.Contains(body, "Try again") {
		t.Fatalf("the worker: %d %s", sw.Code, body)
	}
	if strings.Contains(body, "caches") {
		t.Errorf("it keeps no copy of any page")
	}
}
