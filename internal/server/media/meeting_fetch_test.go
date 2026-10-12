package media_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/meetfetch"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A meeting the app records has its transcript brought once it is over,
// from Zoom here, found by the join link in its where, and given to it as
// its recording with who spoke; the help page says Zoom is connected.
func TestAMeetingsTranscriptIsBroughtFromItsApp(t *testing.T) {
	zoom := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			w.Write([]byte(`{"access_token":"z","expires_in":3600}`))
		case "/meetings/987654321/recordings":
			w.Write([]byte(`{"recording_files":[{"file_type":"TRANSCRIPT","download_url":"http://` + r.Host + `/t"}]}`))
		case "/t":
			w.Write([]byte("WEBVTT\n\n1\n00:00:01.000 --> 00:00:03.000\nAnn Lee: Morning, two things.\n\n2\n00:00:04.000 --> 00:00:06.000\nBen Ortiz: Go on.\n"))
		}
	}))
	defer zoom.Close()
	a, h := newAppWith(t, app.Options{Meetings: meetfetch.At{ZoomOAuth: zoom.URL + "/oauth/token", ZoomAPI: zoom.URL}})
	srv := h.(*server.Server)
	srv.UseAppTokens(filepath.Join(t.TempDir(), "apps.json"))
	t.Setenv("ZOOM_SECRET_TEST", "secret")
	for k, v := range map[string]string{"meetings.zoom_account_id": "acct", "meetings.zoom_client_id": "cid", "meetings.zoom_secret_env": "ZOOM_SECRET_TEST"} {
		if err := a.Workspace.Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
	ended := time.Now().Add(-2 * time.Hour).UTC()
	ev, _ := a.Store.Create("event", map[string]any{"title": "Planning", "where": "https://us02web.zoom.us/j/987654321?pwd=x",
		"starts": ended.Format(time.RFC3339), "ends": ended.Add(time.Hour).Format(time.RFC3339)})
	a.Store.Create("reminder", map[string]any{"title": "Add the recording of Planning", "at": ended.Add(time.Hour).Format(time.RFC3339), "about": chat.RecordAbout(ev.ID)})

	if page := get(t, h, "/help").Body.String(); !strings.Contains(page, "Zoom: connected") {
		t.Error("the help page says Zoom is connected")
	}
	if page := get(t, h, "/t/event/"+ev.ID).Body.String(); !strings.Contains(page, "Zoom is connected, so its transcript is brought here") {
		t.Error("the ended meeting says its transcript is on its way")
	}
	srv.FetchMeetings(time.Now())
	got, _ := a.Store.Get("event", ev.ID)
	id, _ := got.Fields["recording"].(string)
	if id == "" {
		t.Fatal("the transcript is the meeting's recording")
	}
	file, _ := a.Store.Get("file", id)
	if text, _ := file.Fields["text"].(string); !strings.Contains(text, "**Ann Lee:** Morning, two things.") || !strings.Contains(text, "**Ben Ortiz:** Go on.") {
		t.Errorf("with who spoke: %q", text)
	}
}

// A meeting app that answers wrong is said by its job, on Help and to
// agents, until the next round, as every connection in the background is.
func TestAMeetingAppGoingWrongIsSaid(t *testing.T) {
	t.Parallel()
	graph := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down for maintenance", http.StatusServiceUnavailable)
	}))
	defer graph.Close()
	a, h := newAppWith(t, app.Options{Meetings: meetfetch.At{MicrosoftLogin: graph.URL, Graph: graph.URL}})
	srv := h.(*server.Server)
	tokens := filepath.Join(t.TempDir(), "apps.json")
	srv.UseAppTokens(tokens)
	if err := (&meetfetch.Tokens{Path: tokens}).Keep("teams", "a", "r", 3600); err != nil {
		t.Fatal(err)
	}
	if err := a.Workspace.Set("meetings.teams_client_id", "app"); err != nil {
		t.Fatal(err)
	}
	ended := time.Now().Add(-2 * time.Hour).UTC()
	ev, _ := a.Store.Create("event", map[string]any{"title": "Planning", "where": "https://teams.microsoft.com/l/meetup-join/19%3ameeting_abc%40thread.v2/0",
		"starts": ended.Format(time.RFC3339), "ends": ended.Add(time.Hour).Format(time.RFC3339)})
	a.Store.Create("reminder", map[string]any{"title": "Add the recording of Planning", "at": ended.Add(time.Hour).Format(time.RFC3339), "about": chat.RecordAbout(ev.ID)})

	now := time.Now()
	if err := srv.FetchMeetings(now); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("what Microsoft answered is said: %v", err)
	}
	if err := srv.FetchMeetings(now.Add(time.Minute)); err == nil {
		t.Error("and said until the next round")
	}
}
