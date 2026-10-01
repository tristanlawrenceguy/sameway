package meetfetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// Teams signs in by a device code the person types at Microsoft, waits
// while they have not, keeps the sign-in on this computer, and finds a
// meeting's transcript by its join link.
func TestTeamsSignsInAndFetchesATranscript(t *testing.T) {
	var polls atomic.Int32
	join := "https://teams.microsoft.com/l/meetup-join/19%3ameeting_abc%40thread.v2/0"
	ms := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch {
		case strings.HasSuffix(r.URL.Path, "/devicecode"):
			w.Write([]byte(`{"user_code":"ABCD-1234","verification_uri":"https://microsoft.com/devicelogin","device_code":"dev","interval":0,"expires_in":60}`))
		case strings.HasSuffix(r.URL.Path, "/token") && r.Form.Get("grant_type") == "refresh_token":
			w.Write([]byte(`{"access_token":"fresh","refresh_token":"r2","expires_in":3600}`))
		case strings.HasSuffix(r.URL.Path, "/token"):
			if polls.Add(1) < 2 {
				w.WriteHeader(400)
				w.Write([]byte(`{"error":"authorization_pending"}`))
				return
			}
			w.Write([]byte(`{"access_token":"tok","refresh_token":"r1","expires_in":3600}`))
		case r.URL.Path == "/me/onlineMeetings":
			if r.Header.Get("Authorization") != "Bearer tok" || !strings.Contains(r.URL.Query().Get("$filter"), join) {
				w.WriteHeader(403)
				return
			}
			w.Write([]byte(`{"value":[{"id":"m1"}]}`))
		case r.URL.Path == "/me/onlineMeetings/m1/transcripts":
			w.Write([]byte(`{"value":[{"id":"t1"}]}`))
		case r.URL.Path == "/me/onlineMeetings/m1/transcripts/t1/content":
			w.Write([]byte("WEBVTT\n\n00:00:01.000 --> 00:00:02.000\n<v Ann Lee>Morning.</v>\n"))
		default:
			w.WriteHeader(404)
		}
	}))
	defer ms.Close()
	MicrosoftLogin, Graph = ms.URL, ms.URL
	teams := &Teams{ClientID: "app", Tokens: &Tokens{Path: filepath.Join(t.TempDir(), "apps.json")}}
	ctx := context.Background()
	if _, err := teams.Transcript(ctx, join); err != ErrNotConnected {
		t.Errorf("before signing in, it is not connected: %v", err)
	}
	dc, err := teams.Begin(ctx)
	if err != nil || dc.UserCode != "ABCD-1234" {
		t.Fatalf("a code to type at Microsoft: %+v %v", dc, err)
	}
	if err := teams.Finish(ctx, dc); err != nil {
		t.Fatal(err)
	}
	if TeamsJoin("Join: "+join+" (dial-in below)") != join {
		t.Error("the join link is found in the meeting's where")
	}
	vtt, err := teams.Transcript(ctx, join)
	if err != nil || !strings.Contains(string(vtt), "<v Ann Lee>") {
		t.Errorf("the transcript comes with who spoke: %q %v", vtt, err)
	}
}

// Zoom signs in as the account's own app and finds a meeting's transcript
// among its cloud recording's files; with none yet, it says so.
func TestZoomFetchesATranscript(t *testing.T) {
	ready := false
	zoom := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/token":
			if u, p, _ := r.BasicAuth(); u != "cid" || p != "secret" || r.URL.Query().Get("account_id") != "acct" {
				w.WriteHeader(401)
				w.Write([]byte(`{"reason":"Invalid client_id or client_secret"}`))
				return
			}
			w.Write([]byte(`{"access_token":"z","expires_in":3600}`))
		case r.URL.Path == "/meetings/987654321/recordings":
			if !ready {
				w.WriteHeader(404)
				return
			}
			w.Write([]byte(`{"recording_files":[{"file_type":"MP4","download_url":"x"},{"file_type":"TRANSCRIPT","status":"completed","download_url":"` + "http://" + r.Host + `/download/t"}]}`))
		case r.URL.Path == "/download/t":
			w.Write([]byte("WEBVTT\n\n1\n00:00:01.000 --> 00:00:02.000\nAnn Lee: Morning.\n"))
		}
	}))
	defer zoom.Close()
	ZoomOAuth, ZoomAPI = zoom.URL+"/oauth/token", zoom.URL
	z := &Zoom{AccountID: "acct", ClientID: "cid", Secret: "secret"}
	if ZoomMeeting("https://us02web.zoom.us/j/987654321?pwd=abc") != "987654321" {
		t.Error("the meeting number is found in the join link")
	}
	if _, err := z.Transcript(context.Background(), "987654321"); err != ErrNotYet {
		t.Errorf("with no recording yet, it says so: %v", err)
	}
	ready = true
	vtt, err := z.Transcript(context.Background(), "987654321")
	if err != nil || !strings.Contains(string(vtt), "Ann Lee: Morning.") {
		t.Errorf("the transcript: %q %v", vtt, err)
	}
	if _, err := (&Zoom{AccountID: "acct", ClientID: "cid", Secret: "wrong"}).Transcript(context.Background(), "987654321"); err == nil || !strings.Contains(err.Error(), "would not sign in") {
		t.Errorf("a wrong secret says what is wrong: %v", err)
	}
}
