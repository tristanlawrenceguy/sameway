package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// homeWithPhone is a server whose Wi-Fi answers, with the owner's phone
// paired from addr; it says what rang, and has the phone ask for a page.
func homeWithPhone(t *testing.T, a *app.App, addr string, knock func(string) bool) (srv *server.Server, rang *[]string, visit func(), look func(time.Time)) {
	t.Helper()
	srv = server.New(a).WithHomeKnock(knock)
	h := srv.WithFleet(&server.Fleet{Launch: func(string, string) error { return nil }, Exit: func() {},
		LAN: func(bool) error { return nil }, LANBase: func() string { return "http://192.168.1.20:8080" }})
	rang = &[]string{}
	var mu sync.Mutex
	srv.OnRing(func(title, text, url string) { mu.Lock(); *rang = append(*rang, title); mu.Unlock() })
	postForm(t, h, "/phone/on", nil)
	code := regexp.MustCompile(`<strong>([B-Z]{4}-[B-Z]{4})</strong>`).FindStringSubmatch(get(t, h, "/workspaces").Body.String())
	wifi := srv.LAN(h)
	ask := func(path string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.RemoteAddr = addr + ":51000"
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		wifi.ServeHTTP(w, r)
		return w
	}
	var cookie *http.Cookie
	for _, c := range ask("/pair?code="+code[1], nil).Result().Cookies() {
		cookie = c
	}
	visit = func() { ask("/", cookie) }
	look = func(now time.Time) {
		for _, j := range srv.BackgroundJobs() {
			if j.Name == "Phones looked for on the home Wi-Fi" {
				j.Run(context.Background(), now)
			}
		}
		time.Sleep(50 * time.Millisecond) // a ring is told on its own goroutine
		mu.Lock()
		mu.Unlock()
	}
	return srv, rang, visit, look
}

// "Remind me when I get home": the reminder waits, not for a time, but
// for the owner's phone to be back on the home Wi-Fi after a while away,
// seen when it opens Sameway there or when it answers the computer's
// knock. A phone seen for the first time, or a few minutes after it was
// last seen, rings nothing; one back after a quarter of an hour does.
func TestAReminderRingsOnGettingHome(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	now := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	a.Records.Clock = func() time.Time { return now }
	there := false
	srv, rang, visit, look := homeWithPhone(t, a, "192.168.1.50", func(ip string) bool { return there && ip == "192.168.1.50" })
	visit() // at home, seen for the first time
	rec, _ := a.Store.Create("reminder", map[string]any{"title": "Water the plants", "at": now.Format(time.RFC3339), "home": true})
	state := func() string {
		r, _ := a.Store.Get("reminder", rec.ID)
		s, _ := r.Fields["state"].(string)
		return s
	}

	now, there = now.Add(5*time.Minute), true
	look(now)
	if state() != "set" {
		t.Fatalf("still at home, seen again: nothing rings: %s", state())
	}
	now, there = now.Add(time.Hour), false
	srv.Ring(now)
	look(now)
	if state() != "set" {
		t.Fatalf("its time does not ring it, nor the phone being out: %s", state())
	}
	now, there = now.Add(30*time.Minute), true
	look(now)
	if state() != "rang" || len(*rang) != 1 || (*rang)[0] != "Water the plants" {
		t.Errorf("back home, it rings, told as any ring is: %s %v", state(), *rang)
	}
}

// The knock is real: a phone there refuses a knock at a closed port,
// which says it is there, on Windows as anywhere.
func TestAPhoneThereIsKnownByItsRefusal(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	now := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	a.Records.Clock = func() time.Time { return now }
	_, rang, visit, look := homeWithPhone(t, a, "127.0.0.1", nil)
	visit()
	a.Store.Create("reminder", map[string]any{"title": "Bins out", "at": now.Format(time.RFC3339), "home": true})
	now = now.Add(time.Hour)
	look(now)
	if len(*rang) != 1 {
		t.Errorf("this computer, knocked at, refuses: there: %v", *rang)
	}
}
