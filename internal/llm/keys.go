package llm

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// A key is read from the environment variable api_key_env names, and,
// when that is not set, from a small file in the person's own settings
// folder, where a key pasted into Sameway is kept. Setting an environment
// variable is beyond most people; pasting a key is not. The file is never
// in the workspace, so a workspace shared, synced or zipped carries no key.

// KeysPath is the person's keys file. SAMEWAY_KEYS names another, for
// keeping it elsewhere.
func KeysPath() string {
	if p := os.Getenv("SAMEWAY_KEYS"); p != "" {
		return p
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "sameway", "keys.json")
}

// Keys is a keys file, named by the app (workspace.Machine), so a test
// keeps its own; "" is the person's, KeysPath.
type Keys string

func (k Keys) path() string {
	if k == "" {
		return KeysPath()
	}
	return string(k)
}

// Get is the key named: from the environment, else the keys file.
func (k Keys) Get(name string) string {
	if name == "" {
		return ""
	}
	if v := os.Getenv(name); v != "" {
		return v
	}
	return k.read()[name]
}

// Save keeps a key in the keys file under its name, readable by its
// owner alone.
func (k Keys) Save(name, value string) error {
	keys := k.read()
	keys[name] = value
	raw, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(k.path()), 0o700); err != nil {
		return err
	}
	return os.WriteFile(k.path(), raw, 0o600)
}

func (k Keys) read() map[string]string {
	keys := map[string]string{}
	if raw, err := os.ReadFile(k.path()); err == nil {
		json.Unmarshal(raw, &keys)
	}
	return keys
}
