// Package peers keeps copies of one workspace, hosted on different
// computers, the same. Each copy holds every shared field with the stamp
// of its latest write (see store/state.go); keeping two copies in step is
// each telling the other what it has seen and getting back what it has
// not. It runs over the tailnet, between computers the owner has named.
package peers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Message is one side of an exchange: the latest stamp it holds from each
// computer, and stamps the other side lacks.
type Message struct {
	Seen   map[string]string `json:"seen"`
	Stamps []store.Stamp     `json:"stamps"`
	// Present is who is in the workspace on the side that sends it, just
	// now, and where: how people on different computers see each other.
	Present []Presence `json:"present,omitempty"`
}

// Presence is one person in the workspace just now.
type Presence struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Place string `json:"place,omitempty"` // what they are looking at, by name
}

// Answer takes what a peer sent and answers with what it lacks. It says
// how many records the peer's stamps changed here.
func Answer(st *store.Store, in Message) (Message, int, error) {
	n, err := st.Apply(in.Stamps)
	if err != nil {
		return Message{}, 0, err
	}
	seen, err := st.Seen()
	if err != nil {
		return Message{}, n, err
	}
	out, err := st.Since(in.Seen)
	return Message{Seen: seen, Stamps: out}, n, err
}

// With brings this copy and the one on peer into step, both ways: what
// the peer has that this copy lacks, then the other way round. It says
// how many records changed here.
func With(ctx context.Context, c *http.Client, st *store.Store, peer string) (int, error) {
	n, _, err := WithPresence(ctx, c, st, peer, nil)
	return n, err
}

// WithPresence is With that also says who is here, and hears who is
// there.
func WithPresence(ctx context.Context, c *http.Client, st *store.Store, peer string, here []Presence) (int, []Presence, error) {
	mine, err := st.Seen()
	if err != nil {
		return 0, nil, err
	}
	reply, err := post(ctx, c, peer, Message{Seen: mine, Present: here})
	if err != nil {
		return 0, nil, err
	}
	n, err := st.Apply(reply.Stamps)
	if err != nil {
		return n, reply.Present, err
	}
	theirs, err := st.Since(reply.Seen)
	if err != nil || len(theirs) == 0 {
		return n, reply.Present, err
	}
	mine, _ = st.Seen()
	_, err = post(ctx, c, peer, Message{Seen: mine, Stamps: theirs})
	return n, reply.Present, err
}

// Peers reads the peers setting: machine names, comma separated.
func Peers(setting string) []string {
	var out []string
	for _, p := range strings.Split(setting, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func post(ctx context.Context, c *http.Client, peer string, m Message) (Message, error) {
	body, err := json.Marshal(m)
	if err != nil {
		return Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+peer+"/sync", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return Message{}, fmt.Errorf("%s answered %d: %s", peer, res.StatusCode, strings.TrimSpace(string(msg)))
	}
	var out Message
	err = json.NewDecoder(res.Body).Decode(&out)
	return out, err
}
