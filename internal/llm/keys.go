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

// KeysPath is the file pasted keys are kept in. SAMEWAY_KEYS names another,
// for tests.
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

// Key is the key named: from the environment, else the keys file.
func Key(name string) string {
	if name == "" {
		return ""
	}
	if v := os.Getenv(name); v != "" {
		return v
	}
	return readKeys()[name]
}

// SaveKey keeps a key in the keys file under its name, readable by its
// owner alone.
func SaveKey(name, value string) error {
	keys := readKeys()
	keys[name] = value
	raw, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(KeysPath()), 0o700); err != nil {
		return err
	}
	return os.WriteFile(KeysPath(), raw, 0o600)
}

func readKeys() map[string]string {
	keys := map[string]string{}
	if raw, err := os.ReadFile(KeysPath()); err == nil {
		json.Unmarshal(raw, &keys)
	}
	return keys
}
