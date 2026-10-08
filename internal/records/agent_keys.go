package records

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An agent gets in as a person does, by being let in, and gets a key of
// its own: the key names it in the log, where a header naming itself was
// anyone's to write, and says what it may do, so one agent can be given
// less than the owner and taken away without the others. Only the key's
// fingerprint is kept; the key is shown once, where it is made, and never
// passes through the conversation.

// KeyPrefix starts every agent key, so one is told from the MCP token.
const KeyPrefix = "sw_"

func keyPrint(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// LetAgentIn makes a key for an agent, logged as who let it in, and
// returns the key, which is not kept anywhere.
func LetAgentIn(st *store.Store, who Who, name, access string) (string, *store.Record, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, errors.New("an agent needs a name, what the log will call it")
	}
	if access == "" {
		access = Edit
	}
	if access != View && access != Edit && access != Owner {
		return "", nil, fmt.Errorf("access is view, edit or owner, not %q", access)
	}
	if AgentNamed(st, name) != nil {
		return "", nil, fmt.Errorf("there is already an agent called %s; take its key away first, or choose another name", name)
	}
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	key := KeyPrefix + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw))
	// Logged, so it can be seen and taken back (logAs lets agents' keys in).
	rec, _, err := WriteKept(st, who, "created", AgentType, "", map[string]any{"name": name, "access": access, "key": keyPrint(key)})
	if err != nil {
		return "", nil, err
	}
	return key, rec, nil
}

// AgentByKey is the agent a key belongs to, or nil.
func AgentByKey(st *store.Store, key string) *store.Record {
	if !strings.HasPrefix(key, KeyPrefix) {
		return nil
	}
	print := keyPrint(key)
	agents, _ := st.List(AgentType, store.ListOptions{})
	for _, a := range agents {
		if a.Fields["key"] == print {
			return a
		}
	}
	return nil
}

// AgentNamed is the agent of that name, or nil.
func AgentNamed(st *store.Store, name string) *store.Record {
	agents, _ := st.List(AgentType, store.ListOptions{})
	for _, a := range agents {
		if n, _ := a.Fields["name"].(string); strings.EqualFold(n, strings.TrimSpace(name)) {
			return a
		}
	}
	return nil
}

// Agents lists the agents let in, by name.
func Agents(st *store.Store) []*store.Record {
	agents, _ := st.List(AgentType, store.ListOptions{OrderBy: "created_at"})
	return agents
}

// TakeAgentAway removes an agent's key, and so its access, and says it as
// a change with what it was, so undoing it lets the agent back in.
func TakeAgentAway(st *store.Store, name string) (Change, error) {
	a := AgentNamed(st, name)
	if a == nil {
		var names []string
		for _, x := range Agents(st) {
			names = append(names, fmt.Sprint(x.Fields["name"]))
		}
		if len(names) == 0 {
			return Change{}, errors.New("no agent has a key here")
		}
		return Change{}, fmt.Errorf("no agent called %q; those with keys are %s", name, strings.Join(names, ", "))
	}
	if err := st.Delete(AgentType, a.ID); err != nil {
		return Change{}, err
	}
	return Change{Action: "deleted", Component: AgentType, ID: a.ID, Detail: fmt.Sprint(a.Fields["name"]) + "'s key", Before: a.Fields}, nil
}

// Used notes when a key was used, at most once a minute.
func Used(st *store.Store, a *store.Record) {
	if at, ok := a.Fields["last_used"].(string); ok {
		if t, err := time.Parse(time.RFC3339, at); err == nil && time.Since(t) < time.Minute {
			return
		}
	}
	st.Update(AgentType, a.ID, map[string]any{"last_used": time.Now().UTC().Format(time.RFC3339)})
}
