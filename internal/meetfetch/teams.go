// Package meetfetch brings a meeting's transcript from the app that held
// it, once it is over: Microsoft Teams through Microsoft Graph, Zoom
// through its cloud recordings. Each needs the person's own app
// registration and sign-in; a password or a secret never passes through
// here. Teams signs in with a device code the person types at Microsoft;
// Zoom's secret is read from the environment variable its setting names.
package meetfetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Where Microsoft and Zoom answer; tests stand their own in.
var (
	MicrosoftLogin = "https://login.microsoftonline.com"
	Graph          = "https://graph.microsoft.com/v1.0"
)

// teamsScopes are what is asked of Microsoft: to read the person's own
// meetings and their transcripts, and to stay signed in.
const teamsScopes = "offline_access OnlineMeetings.Read OnlineMeetingTranscript.Read.All"

// Teams is a connection to Microsoft Graph as the person.
type Teams struct {
	ClientID, Tenant string
	Tokens           *Tokens // kept on this computer; see tokens.go
	Client           *http.Client
}

// DeviceCode is what the person is shown to sign in: a code to type at an
// address of Microsoft's.
type DeviceCode struct {
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Message         string `json:"message"`
	DeviceCode      string `json:"device_code"`
	Interval        int    `json:"interval"`
	ExpiresIn       int    `json:"expires_in"`
}

func (t *Teams) client() *http.Client {
	if t.Client != nil {
		return t.Client
	}
	return http.DefaultClient
}

func (t *Teams) tenant() string {
	if t.Tenant == "" {
		return "organizations"
	}
	return t.Tenant
}

// Begin asks Microsoft for a code for the person to sign in with.
func (t *Teams) Begin(ctx context.Context) (*DeviceCode, error) {
	if t.ClientID == "" {
		return nil, errors.New("meetings.teams_client_id is not set: it is the application (client) id of an app registered in Microsoft Entra for this")
	}
	var dc DeviceCode
	err := t.post(ctx, MicrosoftLogin+"/"+t.tenant()+"/oauth2/v2.0/devicecode", url.Values{"client_id": {t.ClientID}, "scope": {teamsScopes}}, &dc)
	return &dc, err
}

// Finish waits for the person to sign in with the code, and keeps what
// Microsoft gives to stay signed in.
func (t *Teams) Finish(ctx context.Context, dc *DeviceCode) error {
	wait := time.Duration(max(dc.Interval, 1)) * time.Second
	deadline := time.Now().Add(time.Duration(max(dc.ExpiresIn, 60)) * time.Second)
	for time.Now().Before(deadline) {
		var tok tokenAnswer
		err := t.post(ctx, MicrosoftLogin+"/"+t.tenant()+"/oauth2/v2.0/token", url.Values{"client_id": {t.ClientID},
			"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}, "device_code": {dc.DeviceCode}}, &tok)
		var oe *oauthError
		switch {
		case err == nil:
			return t.Tokens.Keep("teams", tok.AccessToken, tok.RefreshToken, tok.ExpiresIn)
		case errors.As(err, &oe) && oe.Code == "authorization_pending":
		case errors.As(err, &oe) && oe.Code == "slow_down":
			wait += 5 * time.Second
		default:
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
	return errors.New("the code ran out before it was used; connect again")
}

// access is a token Graph takes, refreshed when it has run out.
func (t *Teams) access(ctx context.Context) (string, error) {
	kept, ok := t.Tokens.Get("teams")
	if !ok {
		return "", ErrNotConnected
	}
	if time.Now().Before(kept.Expires.Add(-time.Minute)) {
		return kept.Access, nil
	}
	var tok tokenAnswer
	if err := t.post(ctx, MicrosoftLogin+"/"+t.tenant()+"/oauth2/v2.0/token", url.Values{"client_id": {t.ClientID},
		"grant_type": {"refresh_token"}, "refresh_token": {kept.Refresh}, "scope": {teamsScopes}}, &tok); err != nil {
		return "", fmt.Errorf("Microsoft would not renew the sign-in (%v); connect Teams again", err)
	}
	if tok.RefreshToken == "" {
		tok.RefreshToken = kept.Refresh
	}
	return tok.AccessToken, t.Tokens.Keep("teams", tok.AccessToken, tok.RefreshToken, tok.ExpiresIn)
}

var teamsJoin = regexp.MustCompile(`https://teams\.microsoft\.com/l/meetup-join/[^\s<>"]+`)

// TeamsJoin is the Teams join link in a meeting's where, or "".
func TeamsJoin(where string) string { return teamsJoin.FindString(where) }

// Transcript is a Teams meeting's transcript as WebVTT, with who spoke,
// found by its join link; ErrNotYet when Teams has none for it yet.
func (t *Teams) Transcript(ctx context.Context, join string) ([]byte, error) {
	tok, err := t.access(ctx)
	if err != nil {
		return nil, err
	}
	var meetings struct {
		Value []struct{ ID string } `json:"value"`
	}
	filter := url.QueryEscape("JoinWebUrl eq '" + strings.ReplaceAll(join, "'", "''") + "'")
	if err := t.get(ctx, tok, Graph+"/me/onlineMeetings?$filter="+filter, &meetings); err != nil {
		return nil, err
	}
	if len(meetings.Value) == 0 {
		return nil, errors.New("Teams knows no meeting of yours with that join link; only the organiser's meetings can be read")
	}
	id := meetings.Value[0].ID
	var list struct {
		Value []struct {
			ID      string    `json:"id"`
			Created time.Time `json:"createdDateTime"`
		} `json:"value"`
	}
	if err := t.get(ctx, tok, Graph+"/me/onlineMeetings/"+url.PathEscape(id)+"/transcripts", &list); err != nil {
		return nil, err
	}
	if len(list.Value) == 0 {
		return nil, ErrNotYet
	}
	last := list.Value[len(list.Value)-1]
	return t.raw(ctx, tok, Graph+"/me/onlineMeetings/"+url.PathEscape(id)+"/transcripts/"+url.PathEscape(last.ID)+"/content?$format=text/vtt")
}

type tokenAnswer struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type oauthError struct {
	Code, Description string
}

func (e *oauthError) Error() string { return e.Code + ": " + e.Description }

func (t *Teams) post(ctx context.Context, to string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, to, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return answer(t.client(), req, out)
}

func (t *Teams) get(ctx context.Context, tok, to string, out any) error {
	body, err := t.raw(ctx, tok, to)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (t *Teams) raw(ctx context.Context, tok, to string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, to, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	res, err := t.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if res.StatusCode == http.StatusForbidden && strings.Contains(string(body), "SpeakerAttributionNotAllowed") {
		return nil, errors.New("the organisation's Teams policy keeps speaker names out of transcripts, so Graph will not give it this way")
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("Microsoft Graph answered %s: %s", res.Status, clip(string(body)))
	}
	return body, nil
}

// answer reads a token endpoint's answer, or its error.
func answer(c *http.Client, req *http.Request, out any) error {
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 300 {
		var e struct {
			Error       string `json:"error"`
			Description string `json:"error_description"`
			Reason      string `json:"reason"`
		}
		json.Unmarshal(body, &e)
		if e.Error != "" {
			return &oauthError{Code: e.Error, Description: e.Description + e.Reason}
		}
		return fmt.Errorf("%s answered %s: %s", req.URL.Host, res.Status, clip(string(body)))
	}
	return json.Unmarshal(body, out)
}

func clip(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}

// ErrNotConnected is a meeting app nobody has connected yet.
var ErrNotConnected = errors.New("not connected")

// ErrNotYet is a meeting whose transcript the app has not made yet.
var ErrNotYet = errors.New("no transcript yet")
