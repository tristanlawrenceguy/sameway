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
// as an ask: the owner's own question, what their Sameway put to its
// owner, and the address it gave for it, the mirror of what others ask
// here. Sameway reads each address now and then, and logs what it finds,
// so an action can wait for it: an answer, then each change to what the
// yes made. Of what comes back, only two words are read: the state, from
// a short list, and when it last changed. What it is now is the
// address's to say, read with ask_page.

// sentFor is how long a request is followed before Sameway stops reading.
const sentFor = 30 * 24 * time.Hour

// keepSent keeps each request a question to host put to its owner, from
// what it answered: only addresses on host itself are kept.
func (s *Service) keepSent(host, question string, got []byte) {
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
		s.Apply(records.Op{Type: records.AskType, After: map[string]any{"question": trim.Clip(question, 2000), "asked": trim.Line(r.Asked, 300), "at": u.String(), "state": "waiting"}})
	}
}

// FollowSent reads what became of each request still followed, and logs
// and tells each answer and each change since. It is a background job's
// work.
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
		if r.Fields["state"] == "no" || at == "" || s.Now().Sub(r.CreatedAt) > sentFor {
			continue
		}
		state, changed := s.theirState(ctx, at)
		if state != "yes" && state != "no" || state == r.Fields["state"] && changed == fmt.Sprint(r.Fields["changed"]) {
			continue
		}
		s.answered(r, at, state, changed)
	}
	return nil
}

// theirState is the state their Sameway says a request is in, and when
// what it made last changed, or "".
func (s *Service) theirState(ctx context.Context, at string) (state, changed string) {
	client := &http.Client{Timeout: 30 * time.Second}
	if s.Automate != nil && s.Automate.HTTP != nil {
		client = s.Automate.HTTP
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, at, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer res.Body.Close()
	var out struct {
		State   string `json:"state"`
		Changed string `json:"changed"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 16<<10)).Decode(&out) != nil {
		return "", ""
	}
	return out.State, trim.Line(out.Changed, 40)
}

// answered keeps an answer, or a change to what it made, logs it, so what
// the owner set to happen on it happens, and tells the owner, in the
// conversation and beyond the page.
func (s *Service) answered(r *store.Record, at, state, changed string) {
	done, err := s.Apply(records.Op{Type: records.AskType, ID: r.ID, After: map[string]any{"state": state, "changed": changed}})
	if err != nil {
		return
	}
	u, _ := url.Parse(at)
	asked, _ := r.Fields["asked"].(string)
	title, said := "They answered your request", fmt.Sprintf("%s said yes to your request to %q.", u.Host, asked)
	switch {
	case state == "no":
		said = fmt.Sprintf("%s said no to your request to %q.", u.Host, asked)
	case r.Fields["state"] == "yes" && changed == RequestGone:
		title, said = "They changed what they agreed to", fmt.Sprintf("%s took away what they made for your request to %q.", u.Host, asked)
	case r.Fields["state"] == "yes":
		title, said = "They changed what they agreed to", fmt.Sprintf("%s changed what they made for your request to %q.", u.Host, asked)
	}
	records.Record(s.Store, "system", records.Change{Action: "answered", Component: records.AskType, ID: r.ID, Detail: u.Host, Ops: done})
	s.Say(said)
	if s.Automate != nil && s.Automate.Tell != nil {
		s.Automate.Tell(title, said, "/t/"+records.AskType+"/"+r.ID)
	}
}
