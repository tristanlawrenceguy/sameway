package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// What the owner asks for on someone else's Sameway (ask_page.go) is kept
// as an ask, with the address their Sameway gave for it, the
// mirror of what others ask here. Sameway reads each address now and then
// and tells the owner when they answer, in its own words: of what comes
// back, only the state is read, a word from a short list.

// sentFor is how long a request is followed before Sameway stops reading.
const sentFor = 30 * 24 * time.Hour

// keepSent keeps each request a question to host put to its owner, from
// what it answered: only addresses on host itself are kept.
func (s *Service) keepSent(host string, got []byte) {
	if !s.hasTypes(records.AskType) {
		return
	}
	var out struct {
		Requests []struct {
			Asked  string `json:"asked"`
			Status string `json:"status"`
		} `json:"requests"`
	}
	json.Unmarshal(got, &out)
	for _, r := range out.Requests {
		u, err := url.Parse(r.Status)
		if err != nil || u.Scheme != "https" || u.Host != host || !strings.HasPrefix(u.Path, "/request/") || strings.TrimSpace(r.Asked) == "" {
			continue
		}
		s.Apply(records.Op{Type: records.AskType, After: map[string]any{"asked": trim.Line(r.Asked, 300), "at": u.String(), "state": "waiting"}})
	}
}

// FollowSent reads what became of each request still waiting, and tells
// the owner of each answer. It is a background job's work.
func (s *Service) FollowSent(ctx context.Context) error {
	if !s.hasTypes(records.AskType) {
		return nil
	}
	recs, err := s.Store.List(records.AskType, store.ListOptions{})
	if err != nil {
		return err
	}
	for _, r := range recs {
		at, _ := r.Fields["at"].(string)
		if r.Fields["state"] != "waiting" || at == "" || s.Now().Sub(r.CreatedAt) > sentFor {
			continue
		}
		if state := s.theirState(ctx, at); state == "yes" || state == "no" {
			s.answered(r, at, state)
		}
	}
	return nil
}

// theirState is the state their Sameway says a request is in, or "".
func (s *Service) theirState(ctx context.Context, at string) string {
	client := &http.Client{Timeout: 30 * time.Second}
	if s.Automate != nil && s.Automate.HTTP != nil {
		client = s.Automate.HTTP
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, at, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	var out struct {
		State string `json:"state"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 16<<10)).Decode(&out) != nil {
		return ""
	}
	return out.State
}

// answered keeps an answer and tells the owner, in the conversation and
// beyond the page.
func (s *Service) answered(r *store.Record, at, state string) {
	if _, err := s.Apply(records.Op{Type: records.AskType, ID: r.ID, After: map[string]any{"state": state}}); err != nil {
		return
	}
	u, _ := url.Parse(at)
	asked, _ := r.Fields["asked"].(string)
	said := fmt.Sprintf("%s said yes to your request to %q.", u.Host, asked)
	if state == "no" {
		said = fmt.Sprintf("%s said no to your request to %q.", u.Host, asked)
	}
	s.Say(said)
	if s.Automate != nil && s.Automate.Tell != nil {
		s.Automate.Tell("They answered your request", said, "/t/"+records.AskType+"/"+r.ID)
	}
}
