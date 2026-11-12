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
func homeWithPhone(t *testing.T, a *app.App, addr string, knock func(string) bool) (srv *server.Server, rang func() []string, visit func()) {
	t.Helper()
	srv = server.New(a).WithHomeKnock(knock)
	h := srv.WithFleet(&server.Fleet{Launch: func(string, string) error { return nil }, Exit: func() {},
		LAN: func(bool) error { return nil }, LANBase: func() string { return "http://192.168.1.20:8080" }})
	var told []string
	var mu sync.Mutex
	srv.OnRing(func(title, text, url string) { mu.Lock(); told = append(told, title); mu.Unlock() })
	rang = func() []string {
		time.Sleep(50 * time.Millisecond) // a ring is told on its own goroutine
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), told...)
	}
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
	return srv, rang, visit
}

// lookFor runs the server's knocks every half minute for d, as the
// runner does while the computer is awake, moving the clock on.
func lookFor(srv *server.Server, now *time.Time, d time.Duration) {
	for _, j := range srv.BackgroundJobs() {
		if j.Name != "Phones looked for on the home Wi-Fi" {
			continue
		}
		for end := now.Add(d); now.Before(end); {
			*now = now.Add(30 * time.Second)
			j.Run(context.Background(), *now)
		}
	}
}

// "Remind me when I get home": the reminder waits, not for a time, but
// for the owner's phone to be back on the home Wi-Fi after a while away,
// seen when it opens Sameway there or when it answers the computer's
// knock. A phone at home, or out of reach for a few minutes, rings
// nothing; one back after a quarter of an hour of knocks unanswered does.
func TestAReminderRingsOnGettingHome(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	now := time.Date(2026, 11, 12, 8, 0, 0, 0, time.UTC)
	a.Records.Clock = func() time.Time { return now }
	there := true
	srv, rang, visit := homeWithPhone(t, a, "192.168.1.50", func(ip string) bool { return there && ip == "192.168.1.50" })
	visit() // at home, seen for the first time
	rec, _ := a.Store.Create("reminder", map[string]any{"title": "Water the plants", "at": now.Format(time.RFC3339), "home": true})
	later, _ := a.Store.Create("reminder", map[string]any{"title": "Friday's bins", "at": now.Add(48 * time.Hour).Format(time.RFC3339), "home": true})
	state := func(id string) string {
		r, _ := a.Store.Get("reminder", id)
		s, _ := r.Fields["state"].(string)
		return s
	}

	lookFor(srv, &now, 10*time.Minute)
	there = false
	lookFor(srv, &now, 5*time.Minute)
	there = true
	lookFor(srv, &now, time.Minute)
	if state(rec.ID) != "set" {
		t.Fatalf("out of the router's reach for a few minutes: nothing rings: %s", state(rec.ID))
	}
	there = false
	srv.Ring(now)
	lookFor(srv, &now, time.Hour)
	if state(rec.ID) != "set" {
		t.Fatalf("its time does not ring it, nor the phone being out: %s", state(rec.ID))
	}
	there = true
	lookFor(srv, &now, time.Minute)
	if state(rec.ID) != "rang" || len(rang()) != 1 || rang()[0] != "Water the plants" {
		t.Errorf("back home, it rings, told as any ring is: %s %v", state(rec.ID), rang())
	}
	if state(later.ID) != "set" {
		t.Errorf("one for getting home after Friday waits for Friday: %s", state(later.ID))
	}
}

// The computer asleep, off or restarted is not looking, so a phone that
// was at home all along rings nothing when it wakes; and the knocks go
// on after a restart, at the address the phone last had.
func TestGettingHomeIsOnlyWhatTheComputerSaw(t *testing.T) {
	t.Parallel()
	a, _ := newApp(t)
	now := time.Date(2026, 11, 12, 22, 0, 0, 0, time.UTC)
	a.Records.Clock = func() time.Time { return now }
	there := true
	knock := func(ip string) bool { return there && ip == "192.168.1.50" }
	srv, rang, visit := homeWithPhone(t, a, "192.168.1.50", knock)
	visit()
	rec, _ := a.Store.Create("reminder", map[string]any{"title": "Bins out", "at": now.Format(time.RFC3339), "home": true})
	lookFor(srv, &now, 5*time.Minute)

	now = now.Add(9 * time.Hour) // the computer slept the night
	lookFor(srv, &now, time.Minute)
	visit()
	if got := rang(); len(got) != 0 {
		t.Fatalf("at home all night, the computer waking rings nothing: %v", got)
	}

	again, rangAgain, _ := homeWithPhone(t, a, "192.168.1.60", knock) // Sameway started again
	there = false
	lookFor(again, &now, 20*time.Minute)
	there = true
	lookFor(again, &now, time.Minute)
	r, _ := a.Store.Get("reminder", rec.ID)
	if r.Fields["state"] != "rang" || len(rangAgain()) != 1 {
		t.Errorf("after a restart the phone is knocked for where it was, and back, rings: %v %v", r.Fields["state"], rangAgain())
	}

	knocked := false
	gone := server.New(a).WithHomeKnock(func(string) bool { knocked = true; return true })
	a.Store.SetMeta("lan:devices", "[]") // every phone removed on Workspaces
	lookFor(gone, &now, time.Minute)
	if knocked {
		t.Errorf("a phone removed is knocked at no more")
	}
}
