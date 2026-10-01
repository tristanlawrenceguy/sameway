package meetfetch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Tokens are a meeting app's sign-in, kept in the person's own settings
// folder on this computer, readable by them alone: never in a workspace,
// which may be shared or kept in step with other computers.
type Tokens struct {
	Path string
	mu   sync.Mutex
}

// Kept is one app's sign-in.
type Kept struct {
	Access  string    `json:"access"`
	Refresh string    `json:"refresh"`
	Expires time.Time `json:"expires"`
}

// TokensHere is where this computer keeps them.
func TokensHere() *Tokens {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return &Tokens{Path: filepath.Join(dir, "sameway", "meeting-apps.json")}
}

func (t *Tokens) read() map[string]Kept {
	all := map[string]Kept{}
	if data, err := os.ReadFile(t.Path); err == nil {
		json.Unmarshal(data, &all)
	}
	return all
}

// Get is one app's sign-in.
func (t *Tokens) Get(app string) (Kept, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	k, ok := t.read()[app]
	return k, ok && k.Refresh != ""
}

// Keep keeps one app's sign-in.
func (t *Tokens) Keep(app, access, refresh string, expiresIn int) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	all := t.read()
	all[app] = Kept{Access: access, Refresh: refresh, Expires: time.Now().Add(time.Duration(expiresIn) * time.Second)}
	data, _ := json.MarshalIndent(all, "", "  ")
	if err := os.MkdirAll(filepath.Dir(t.Path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(t.Path, data, 0o600)
}

// Forget signs one app out.
func (t *Tokens) Forget(app string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	all := t.read()
	delete(all, app)
	data, _ := json.MarshalIndent(all, "", "  ")
	return os.WriteFile(t.Path, data, 0o600)
}
