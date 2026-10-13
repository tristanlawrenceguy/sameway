package server_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/runner"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A workspace's days are its person's, in its time zone (ui.zone), not
// the computer's: hosted on a server in UTC, or somewhere else, Today,
// tomorrow, a repeat and the page's own day are the person's. The zones
// are far from UTC, one ahead and one behind, so a tenth of a second
// before or after their midnight the computer's day is another.
func TestTheWorkspacesZoneIsItsDays(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"Pacific/Kiritimati", "Pacific/Pago_Pago"} {
		zone, err := time.LoadLocation(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, at := range []time.Time{
			time.Date(2026, 3, 11, 23, 59, 59, 9e8, zone),
			time.Date(2026, 3, 12, 0, 0, 0, 1e8, zone),
		} {
			t.Run(name+at.Format(" 15:04:05.0"), func(t *testing.T) {
				t.Parallel()
				daysAreTheZones(t, name, at)
			})
		}
	}
}

func daysAreTheZones(t *testing.T, name string, at time.Time) {
	// The computer's clock says the moment in UTC, as a server's does.
	a, h := newAppWith(t, app.Options{Clock: func() time.Time { return at.UTC() }})
	if err := a.Workspace.Set("ui.zone", name); err != nil {
		t.Fatal(err)
	}
	today, tomorrow := at.Format("2006-01-02"), at.AddDate(0, 0, 1).Format("2006-01-02")
	_, off := at.Zone()

	page := get(t, h, "/today").Body.String()
	if !strings.Contains(page, fmt.Sprintf(`data-zone="%d" data-today="%s"`, off/60, today)) {
		t.Errorf("the page says the workspace's day and zone, %s and %d\n%.600s", today, off/60, page)
	}
	if !strings.Contains(page, `datetime="`+today+`"`) {
		t.Errorf("Today is %s, the workspace's day\n%.3000s", today, page)
	}

	var due, next, rep struct{ ID string }
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Due today", "due": "today"}), &due)
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Due later", "due": "tomorrow"}), &next)
	if rec, _ := a.Store.Get("task", due.ID); rec.Fields["due"] != today+"T00:00:00Z" {
		t.Errorf("today through the API is %s, got %v", today, rec.Fields["due"])
	}
	if rec, _ := a.Store.Get("task", next.ID); rec.Fields["due"] != tomorrow+"T00:00:00Z" {
		t.Errorf("tomorrow through the API is %s, got %v", tomorrow, rec.Fields["due"])
	}
	page = get(t, h, "/today").Body.String()
	if !strings.Contains(page, ">Due today<") || strings.Contains(page, ">Due later<") {
		t.Errorf("Today lists what is due on the workspace's day, not the next\n%.3000s", page)
	}

	// A daily repeat at nine, finished, is due at nine the next day there.
	nine := time.Date(at.Year(), at.Month(), at.Day(), 9, 0, 0, 0, at.Location())
	decode(t, postJSON(t, h, http.MethodPost, "/api/task", map[string]any{"title": "Water", "due": nine.UTC().Format(time.RFC3339), "repeat": "every day"}), &rep)
	wantStatus(t, postJSON(t, h, http.MethodPatch, "/api/task/"+rep.ID, map[string]any{"done": true}), http.StatusOK)
	if rec, _ := a.Store.Get("task", rep.ID); rec.Fields["due"] != nine.AddDate(0, 0, 1).UTC().Format(time.RFC3339) {
		t.Errorf("a daily repeat at nine is due at nine there the next day, %v, got %v", nine.AddDate(0, 0, 1), rec.Fields["due"])
	}
}

// The owner sets the zone on Help, by a place's name; a name that is not
// one is refused with how to give it, and the change can be undone.
func TestTheOwnerSetsTheZone(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	if page := get(t, h, "/help").Body.String(); !strings.Contains(page, "Time zone: this computer&#39;s time.") {
		t.Errorf("Help says the zone is the computer's until chosen\n%.4000s", page)
	}
	if body := after(t, h, postForm(t, h, "/zone", url.Values{"zone": {"Madrid"}})).Body.String(); !strings.Contains(body, "such as Europe/Madrid") {
		t.Errorf("a zone that is not one is refused with how to give it\n%.4000s", body)
	}
	body := after(t, h, postForm(t, h, "/zone", url.Values{"zone": {"Europe/Madrid"}})).Body.String()
	if !strings.Contains(body, "Times are now Madrid time") || a.Workspace.Config.UI.Zone != "Europe/Madrid" {
		t.Errorf("the zone is set and said: %q\n%.4000s", a.Workspace.Config.UI.Zone, body)
	}
	if loc := a.Now().Location().String(); loc != "Europe/Madrid" {
		t.Errorf("the app's clock follows at once, got %s", loc)
	}
}

// The morning brief comes at its time in the workspace's zone, whatever
// zone the computer's clock is in: 07:30 in Tokyo is 22:30 the day before
// in UTC, either side of the computer's midnight.
func TestTheBriefComesAtTheZonesTime(t *testing.T) {
	t.Parallel()
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	f := runner.NewFake(time.Date(2026, 10, 10, 7, 0, 0, 0, tokyo).UTC())
	a, h := newAppWith(t, app.Options{Clock: f.Now, After: f.After})
	for key, value := range map[string]string{"ui.zone": "Asia/Tokyo", "brief.at": "07:30"} {
		if err := a.Workspace.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}
	a.Store.Create("task", map[string]any{"title": "Pay rent", "due": "2026-10-10"}) // Tokyo's today
	srv := h.(*server.Server)
	told := make(chan string, 8)
	srv.OnRing(func(title, _, _ string) { told <- title })
	for _, j := range srv.BackgroundJobs() {
		if j.Name == "Morning brief" {
			a.Jobs.Add(j)
		}
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	a.Jobs.Start(ctx)
	f.BlockUntil(1)
	for _, st := range a.Jobs.Status() {
		if want := time.Date(2026, 10, 10, 7, 30, 0, 0, tokyo); !st.NextRun.Equal(want) {
			t.Errorf("the brief is next at 07:30 in Tokyo, %v, got %v", want, st.NextRun)
		}
	}
	f.Advance(30 * time.Minute)
	select {
	case got := <-told:
		if !strings.HasPrefix(got, "Today: 1 task") {
			t.Errorf("the brief says Tokyo's day, got %q", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no brief at 07:30 in Tokyo")
	}
}

// With no zone chosen, the owner's pages let the script offer their
// browser's (36-days.js); a viewer's never do, and once one is chosen,
// even this computer's, nobody is asked again.
func TestOnlyTheOwnerIsOfferedTheirZone(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	viewer := records.Visitor{Name: "Vi", Login: "vi@example.com", Access: records.View}
	if page := get(t, h, "/today").Body.String(); !strings.Contains(page, " data-zone-ask") {
		t.Error("the owner may be offered their zone while none is chosen")
	}
	if page := as(t, h, viewer, http.MethodGet, "/today", "", "").Body.String(); strings.Contains(page, "data-zone-ask") {
		t.Error("a viewer is never offered a zone for the workspace")
	}
	if res := as(t, h, viewer, http.MethodPost, "/zone", "zone=Europe/Madrid", "application/x-www-form-urlencoded"); res.Code != http.StatusForbidden || a.Workspace.Config.UI.Zone != "" {
		t.Errorf("a viewer cannot set it: %d %q", res.Code, a.Workspace.Config.UI.Zone)
	}
	wantStatus(t, postForm(t, h, "/zone", url.Values{"zone": {"Local"}}), http.StatusSeeOther)
	if page := get(t, h, "/today").Body.String(); strings.Contains(page, "data-zone-ask") {
		t.Error("kept as this computer's, the owner is not asked again")
	}
}
