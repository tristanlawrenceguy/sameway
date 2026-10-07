package server

import (
	"html/template"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What the help page says of the meeting apps, for the owner: connected,
// waiting for a code to be typed, or ready to connect, in out; what to set
// up, which is an administrator's work in Microsoft's and Zoom's own
// consoles, in setup, which the page keeps closed until asked.
func (s *Server) appLines() (out, setup []string) {
	esc := template.HTMLEscapeString
	s.apps.mu.Lock()
	code, failed := s.apps.code, s.apps.failed
	s.apps.mu.Unlock()
	switch {
	case s.app.Workspace.Config.Meetings.TeamsClientID == "":
		setup = append(setup, "Teams: transcripts are not brought from it. To have them brought once a meeting you organise is over, register an app in Microsoft Entra (public client flows on, with the delegated permissions OnlineMeetings.Read and OnlineMeetingTranscript.Read.All, which your administrator may need to grant) and ask the assistant to set meetings.teams_client_id to its client id.")
	case s.teamsConnected():
		out = append(out, "Teams: connected. The transcript of a meeting you organise, recorded in Teams, is brought once it is over.")
	case code != nil:
		out = append(out, "Teams: type <strong>"+esc(code.UserCode)+"</strong> at "+esc(code.VerificationURI)+" and sign in there; your password goes to Microsoft, never here.")
	default:
		why := ""
		if failed != "" {
			why = " The last try ended: " + esc(failed) + "."
		}
		out = append(out, "Teams: ready to connect."+why+` <form method="post" action="/meetings/teams/connect">`+
			string(s.component("button", map[string]any{"label": "Connect Teams", "type": "submit", "variant": "secondary"}))+`</form>`)
	}
	c := s.app.Workspace.Config.Meetings
	switch {
	case s.zoom().Ready():
		out = append(out, "Zoom: connected. The transcript of a meeting recorded to Zoom's cloud is brought once it is over.")
	case c.ZoomAccountID != "" && c.ZoomClientID != "":
		out = append(out, "Zoom: set up, but the environment variable named in meetings.zoom_secret_env holds no secret where Sameway runs.")
	default:
		setup = append(setup, "Zoom: transcripts are not brought from it. To have them brought, make a Server-to-Server OAuth app in Zoom's marketplace with the cloud recording scope, put its client secret in an environment variable, and ask the assistant to set meetings.zoom_account_id, meetings.zoom_client_id and meetings.zoom_secret_env.")
	}
	return out, setup
}

// fetchSaid is what an ended meeting's page says when its app will bring
// the transcript: which app, and that it comes by itself.
func (s *Server) fetchSaid(ev *store.Record) string {
	where := str(ev.Fields["where"], "")
	switch {
	case strings.Contains(where, "teams.microsoft.com/l/meetup-join/") && s.teamsConnected():
		return " Teams is connected, so its transcript is brought here by itself once Teams has made it; or add it now."
	case strings.Contains(where, "zoom.us/j/") && s.zoom().Ready():
		return " Zoom is connected, so its transcript is brought here by itself once Zoom has made it; or add it now."
	}
	return ""
}
