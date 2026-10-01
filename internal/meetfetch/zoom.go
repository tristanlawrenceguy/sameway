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
)

// Where Zoom answers; tests stand their own in.
var (
	ZoomOAuth = "https://zoom.us/oauth/token"
	ZoomAPI   = "https://api.zoom.us/v2"
)

// Zoom is a Server-to-Server OAuth app in the person's Zoom account, with
// cloud recording and audio transcripts on. Its secret is never kept here:
// it is read from the environment each time.
type Zoom struct {
	AccountID, ClientID, Secret string
	Client                      *http.Client
}

func (z *Zoom) client() *http.Client {
	if z.Client != nil {
		return z.Client
	}
	return http.DefaultClient
}

// Ready says whether everything Zoom needs is set.
func (z *Zoom) Ready() bool { return z.AccountID != "" && z.ClientID != "" && z.Secret != "" }

var zoomJoin = regexp.MustCompile(`https://[\w.-]*zoom\.us/j/(\d{9,12})`)

// ZoomMeeting is the meeting number in a Zoom join link, or "".
func ZoomMeeting(where string) string {
	if m := zoomJoin.FindStringSubmatch(where); m != nil {
		return m[1]
	}
	return ""
}

func (z *Zoom) token(ctx context.Context) (string, error) {
	if !z.Ready() {
		return "", ErrNotConnected
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ZoomOAuth+"?grant_type=account_credentials&account_id="+url.QueryEscape(z.AccountID), nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(z.ClientID, z.Secret)
	var tok tokenAnswer
	if err := answer(z.client(), req, &tok); err != nil {
		return "", fmt.Errorf("Zoom would not sign in with the account id, client id and secret set (%v)", err)
	}
	return tok.AccessToken, nil
}

// Transcript is a Zoom meeting's audio transcript as WebVTT, by its
// number; ErrNotYet when Zoom has none for it yet.
func (z *Zoom) Transcript(ctx context.Context, meeting string) ([]byte, error) {
	tok, err := z.token(ctx)
	if err != nil {
		return nil, err
	}
	body, status, err := z.get(ctx, tok, ZoomAPI+"/meetings/"+url.PathEscape(meeting)+"/recordings")
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, ErrNotYet // no recording yet, or none was made
	}
	if status >= 300 {
		return nil, fmt.Errorf("Zoom answered %d: %s", status, clip(string(body)))
	}
	var rec struct {
		Files []struct {
			Type     string `json:"file_type"`
			Download string `json:"download_url"`
			Status   string `json:"status"`
		} `json:"recording_files"`
	}
	if err := json.Unmarshal(body, &rec); err != nil {
		return nil, err
	}
	for _, f := range rec.Files {
		if strings.EqualFold(f.Type, "TRANSCRIPT") && (f.Status == "" || strings.EqualFold(f.Status, "completed")) {
			vtt, status, err := z.get(ctx, tok, f.Download)
			if err != nil {
				return nil, err
			}
			if status >= 300 {
				return nil, fmt.Errorf("Zoom answered %d for the transcript", status)
			}
			return vtt, nil
		}
	}
	return nil, ErrNotYet
}

func (z *Zoom) get(ctx context.Context, tok, to string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, to, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	res, err := z.client().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return nil, 0, errors.New("the transcript did not arrive whole")
	}
	return body, res.StatusCode, nil
}
