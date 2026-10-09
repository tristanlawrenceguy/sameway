package media

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/meetfetch"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// A meeting the app records (record_meeting, how app) has its transcript
// brought from Teams or Zoom once it is over, when the person has
// connected that app: every few minutes for a day after it ends, found by
// the join link in its where, kept as a file and given to the meeting as
// its recording, with who spoke. Connecting is the owner's: Teams by a
// code typed at Microsoft, Zoom by settings and an environment variable.

type meetingApps struct {
	mu     sync.Mutex
	tokens *meetfetch.Tokens
	code   *meetfetch.DeviceCode // a Teams sign-in being waited for
	failed string                // how the last sign-in ended, when badly
	tried  map[string]time.Time  // meeting → when it was last tried
	last   time.Time             // the last round
}

// fetchEvery is how often a round looks; fetchFor how long after a
// meeting it keeps looking.
const (
	fetchEvery = 5 * time.Minute
	fetchFor   = 24 * time.Hour
)

func (s *Service) appTokens() *meetfetch.Tokens {
	s.apps.mu.Lock()
	defer s.apps.mu.Unlock()
	if s.apps.tokens == nil {
		s.apps.tokens = meetfetch.TokensHere()
	}
	return s.apps.tokens
}

// UseAppTokens keeps the meeting apps' sign-ins at path, for tests.
func (s *Service) UseAppTokens(path string) { s.apps.tokens = &meetfetch.Tokens{Path: path} }

func (s *Service) teams() *meetfetch.Teams {
	c := s.app.Workspace.Config.Meetings
	return &meetfetch.Teams{ClientID: c.TeamsClientID, Tenant: c.TeamsTenant, Tokens: s.appTokens()}
}

func (s *Service) zoom() *meetfetch.Zoom {
	c := s.app.Workspace.Config.Meetings
	secret := ""
	if c.ZoomSecretEnv != "" {
		secret = os.Getenv(c.ZoomSecretEnv)
	}
	return &meetfetch.Zoom{AccountID: c.ZoomAccountID, ClientID: c.ZoomClientID, Secret: secret}
}

func (s *Service) teamsConnected() bool {
	_, ok := s.appTokens().Get("teams")
	return ok && s.app.Workspace.Config.Meetings.TeamsClientID != ""
}

// FetchMeetings looks, at most every few minutes, for meetings over without
// their transcript that an app may now have.
func (s *Service) FetchMeetings(now time.Time) {
	s.apps.mu.Lock()
	if now.Sub(s.apps.last) < fetchEvery {
		s.apps.mu.Unlock()
		return
	}
	s.apps.last = now
	s.apps.mu.Unlock()
	teams, zoom := s.teamsConnected(), s.zoom().Ready()
	if !teams && !zoom {
		return
	}
	et, ok := s.app.Types.Get(records.EventType)
	if !ok {
		return
	}
	recent, _ := query.Filter(s.app.Store, et, []string{"recording=", "starts>=-2d"}, "starts", 0, now)
	for _, ev := range recent {
		if !s.wantsApp(ev, now) {
			continue
		}
		where, _ := ev.Fields["where"].(string)
		join, zoomID := meetfetch.TeamsJoin(where), meetfetch.ZoomMeeting(where)
		var vtt []byte
		var err error
		switch {
		case join != "" && teams:
			vtt, err = s.teams().Transcript(context.Background(), join)
		case zoomID != "" && zoom:
			vtt, err = s.zoom().Transcript(context.Background(), zoomID)
		default:
			continue
		}
		s.apps.mu.Lock()
		if s.apps.tried == nil {
			s.apps.tried = map[string]time.Time{}
		}
		s.apps.tried[ev.ID] = now
		s.apps.mu.Unlock()
		switch {
		case errors.Is(err, meetfetch.ErrNotYet), errors.Is(err, meetfetch.ErrNotConnected):
		case err != nil:
			log.Printf("meetings: %s: %v", ev.ID, err)
		default:
			s.keepFetched(ev, vtt, join != "")
		}
	}
}

// wantsApp says whether a meeting is over, recently, without a recording,
// and was asked to have the app's brought.
func (s *Service) wantsApp(ev *store.Record, now time.Time) bool {
	if r, _ := ev.Fields["recording"].(string); r != "" {
		return false
	}
	st, err := time.Parse(time.RFC3339, str(ev.Fields["starts"], ""))
	if err != nil {
		return false
	}
	end := st.Add(chat.MeetingLength(ev))
	if now.Before(end) || now.Sub(end) > fetchFor {
		return false
	}
	for _, r := range chat.RemindersAbout(s.app.Store, chat.RecordAbout(ev.ID)) {
		if strings.HasPrefix(str(r.Fields["title"], ""), "Add the recording of") {
			return true
		}
	}
	return false
}

// keepFetched keeps a transcript brought from an app and gives it to its
// meeting, as the system's change, logged and undone like any other.
func (s *Service) keepFetched(ev *store.Record, vtt []byte, teams bool) {
	et, _ := s.app.Types.Get(records.EventType)
	title, from := s.Title(et, ev), "Zoom"
	if teams {
		from = "Teams"
	}
	who := records.Who{Actor: "system", Via: "from " + from}
	rec, path, err := s.KeepFile(who, bytes.NewReader(vtt), title+" transcript.vtt", "Transcript of "+title, "")
	if err != nil {
		log.Printf("meetings: keeping %s: %v", title, err)
		return
	}
	s.ReadKept(rec.ID, title+" transcript.vtt", path, true)
	if _, _, err := records.WriteAs(s.app.Store, who, "updated", records.EventType, ev.ID, map[string]any{"recording": rec.ID}); err != nil {
		log.Printf("meetings: giving %s its transcript: %v", title, err)
	}
	s.Changed()
}

// teamsConnect is the owner's press that signs in to Teams: Microsoft
// gives a code, the page shows it, and the person types it at Microsoft
// while this waits for them.
func (s *Service) teamsConnect(w http.ResponseWriter, r *http.Request) {
	teams := s.teams()
	code, err := teams.Begin(r.Context())
	if err != nil {
		s.Tell(w, r, web.Outcome{Failed: true, Title: "Teams not connected", Text: err.Error()}, "/help")
		return
	}
	s.apps.mu.Lock()
	s.apps.code, s.apps.failed = code, ""
	s.apps.mu.Unlock()
	go func() {
		err := teams.Finish(context.Background(), code)
		s.apps.mu.Lock()
		s.apps.code = nil
		if err != nil {
			s.apps.failed = err.Error()
		}
		s.apps.mu.Unlock()
		if err == nil {
			records.Record(s.app.Store, "system", records.Change{Action: "added", Detail: "a connection to Teams, for bringing meeting transcripts"})
		}
		s.Changed()
	}()
	s.Tell(w, r, web.Outcome{Title: "Type " + code.UserCode + " at " + code.VerificationURI, Text: "Sign in there with your work account; this page says when Teams is connected. Your password goes to Microsoft, never here."}, "/help")
}
