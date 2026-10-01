package server_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/meetfetch"
	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A meeting the app records has its transcript brought once it is over,
// from Zoom here, found by the join link in its where, and given to it as
// its recording with who spoke; the help page says Zoom is connected.
func TestAMeetingsTranscriptIsBroughtFromItsApp(t *testing.T) {
	a, h := newApp(t)
	srv := h.(*server.Server)
	srv.UseAppTokens(filepath.Join(t.TempDir(), "apps.json"))
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
	meetfetch.ZoomOAuth, meetfetch.ZoomAPI = zoom.URL+"/oauth/token", zoom.URL
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
