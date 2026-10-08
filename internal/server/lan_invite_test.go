package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// Someone else on the Wi-Fi is invited by name: the link works once and
// lets their device in as them, with what they may do; one who may look
// cannot change anything; taking their access away shuts the device out.
func TestSomeoneOnTheWiFiIsInvitedByName(t *testing.T) {
	a, _ := newApp(t)
	srv := server.New(a)
	h := srv.WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() {},
		LAN:     func(bool) error { return nil },
		LANBase: func() string { return "http://192.168.1.20:8080" }})
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, "Make an invite") {
		t.Fatalf("Workspaces offers an invite: %s", truncate(page))
	}
	res := postForm(t, h, "/phone/invite", url.Values{"name": {"Ana"}, "access": {"view"}})
	m := regexp.MustCompile(`http://192\.168\.1\.20:8080/pair\?invite=([0-9a-f]+)`).FindStringSubmatch(res.Body.String())
	if m == nil || !strings.Contains(res.Body.String(), "Code to scan") {
		t.Fatalf("the invite's link and code: %d %s", res.Code, truncate(res.Body.String()))
	}
	ana := a.Chat.PersonByName("ana")
	if ana == nil || ana.Fields["access"] != chat.View {
		t.Fatalf("Ana may look: %v", ana)
	}
	wifi := srv.LAN(h)
	ask := func(method, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		r.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14)")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		wifi.ServeHTTP(w, r)
		return w
	}
	joined := ask("GET", "/pair?invite="+m[1], nil)
	cookies := joined.Result().Cookies()
	if joined.Code != http.StatusSeeOther || len(cookies) == 0 {
		t.Fatalf("the invite lets the device in: %d", joined.Code)
	}
	cookie := cookies[0]
	if res := ask("GET", "/pair?invite="+m[1], nil); res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "used") {
		t.Error("an invite works once")
	}
	if res := ask("GET", "/t/note", cookie); res.Code != http.StatusOK {
		t.Errorf("Ana reads: %d", res.Code)
	}
	if res := ask("POST", "/t/note/add", cookie); res.Code != http.StatusForbidden {
		t.Errorf("one who may look changes nothing: %d", res.Code)
	}
	if res := ask("GET", "/workspaces", cookie); strings.Contains(res.Body.String(), "Make an invite") {
		t.Error("only the owner invites")
	}
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, "Ana&#39;s Android phone, may look") {
		t.Errorf("the owner sees Ana's phone and what they may do: %s", truncate(page))
	}
	// Edit, then none.
	postForm(t, h, "/phone/invite", url.Values{"name": {"Ana"}, "access": {"edit"}})
	if res := ask("POST", "/t/note/add", cookie); res.Code != http.StatusSeeOther {
		t.Errorf("one who may edit adds: %d", res.Code)
	}
	if _, err := a.Store.Update(chat.PersonType, ana.ID, map[string]any{"access": ""}); err != nil {
		t.Fatal(err)
	}
	if res := ask("GET", "/", cookie); res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "no longer opens") {
		t.Errorf("access taken away shuts the device out: %d", res.Code)
	}
}
