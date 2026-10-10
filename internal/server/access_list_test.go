package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// Who can open this lists everyone let in, however they came, with what
// they may do and how they come in; Take away access shuts every device
// of theirs out and their invite with it, and Undo lets them back.
func TestWhoCanOpenThisListsEveryoneAndTakesItBack(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	h := srv.WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() {},
		LAN:     func(bool) error { return nil },
		LANBase: func() string { return "http://192.168.1.20:8080" }})
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, ">Who can open this<") || !strings.Contains(page, "Only you.") {
		t.Fatalf("with nobody let in, the list says only you: %s", truncate(page))
	}
	if _, err := a.Store.Create(records.PersonType, map[string]any{"name": "Bob", "email": "bob@example.com", "access": records.Edit}); err != nil {
		t.Fatal(err)
	}
	res := postForm(t, h, "/phone/invite", url.Values{"name": {"Ana"}, "access": {"view"}})
	code := regexp.MustCompile(`/pair\?invite=([0-9a-f]+)`).FindStringSubmatch(res.Body.String())
	page := get(t, h, "/workspaces").Body.String()
	for _, want := range []string{"<strong>Ana</strong> may look", "An invite not opened yet, until", "<strong>Bob</strong> may edit", "On Tailscale as bob@example.com", "Take away access"} {
		if !strings.Contains(page, want) {
			t.Errorf("the list says %q: %s", want, truncate(page))
		}
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
	cookie := ask("/pair?invite="+code[1], nil).Result().Cookies()[0]
	if page := get(t, h, "/workspaces").Body.String(); !strings.Contains(page, "1 device on the Wi-Fi") {
		t.Errorf("Ana's phone is said under her name: %s", truncate(page))
	}
	ana := a.Chat.PersonByName("Ana")
	took := postForm(t, h, "/access/take", url.Values{"id": {ana.ID}})
	if took.Code != http.StatusSeeOther {
		t.Fatalf("take away access: %d %s", took.Code, truncate(took.Body.String()))
	}
	if res := ask("/", cookie); res.Code != http.StatusForbidden {
		t.Errorf("Ana's phone is shut out at once: %d", res.Code)
	}
	if page := get(t, h, "/workspaces").Body.String(); strings.Contains(page, "<strong>Ana</strong>") {
		t.Error("Ana is no longer listed")
	}
	entry := lastActivity(t, a, "took access from")
	postForm(t, h, "/activity/"+entry+"/undo", nil)
	if res := ask("/", cookie); res.Code != http.StatusOK {
		t.Errorf("Undo lets Ana's phone back in: %d", res.Code)
	}
}

// An invite not yet opened is taken back with the access: opened later,
// it says so and lets nothing in.
func TestAnInviteIsTakenBackWithTheAccess(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	srv := server.New(a)
	h := srv.WithFleet(&server.Fleet{Launch: func(dir, addr string) error { return nil }, Exit: func() {},
		LAN:     func(bool) error { return nil },
		LANBase: func() string { return "http://192.168.1.20:8080" }})
	res := postForm(t, h, "/phone/invite", url.Values{"name": {"Ana"}, "access": {"edit"}})
	code := regexp.MustCompile(`/pair\?invite=([0-9a-f]+)`).FindStringSubmatch(res.Body.String())
	postForm(t, h, "/access/take", url.Values{"id": {a.Chat.PersonByName("Ana").ID}})
	w := httptest.NewRecorder()
	srv.LAN(h).ServeHTTP(w, httptest.NewRequest("GET", "/pair?invite="+code[1], nil))
	if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "taken back") || len(w.Result().Cookies()) > 0 {
		t.Errorf("the invite no longer works: %d %s", w.Code, w.Body.String())
	}
	if page := get(t, h, "/workspaces").Body.String(); strings.Contains(page, "An invite not opened yet") {
		t.Error("the invite is no longer listed")
	}
}

// lastActivity is the newest log entry with that action.
func lastActivity(t *testing.T, a *app.App, action string) string {
	t.Helper()
	for _, e := range a.Records.Recent(10) {
		if e.Fields["action"] == action {
			return e.ID
		}
	}
	t.Fatalf("no %q in the log", action)
	return ""
}
