package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// The daily copies are kept on this computer, and a broken or lost laptop
// takes them with it. Most people already have a folder their cloud keeps
// (OneDrive, Dropbox, iCloud Drive, Google Drive): a copy put there is
// off the computer within minutes, by the app they already trust, and on
// their next computer as soon as they sign in. CloudFolders finds them.

// Cloud is a folder a cloud service keeps in step with elsewhere.
type Cloud struct {
	Name string `json:"name"`
	Dir  string `json:"dir"`
}

// CloudFolders are the cloud folders on this computer, as their apps put
// them.
func CloudFolders() []Cloud {
	home, _ := os.UserHomeDir()
	var found []Cloud
	add := func(name, dir string) {
		if dir == "" {
			return
		}
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			for _, c := range found {
				if c.Dir == dir {
					return
				}
			}
			found = append(found, Cloud{name, dir})
		}
	}
	add("OneDrive", os.Getenv("OneDrive"))
	add("OneDrive", filepath.Join(home, "OneDrive"))
	add("Dropbox", dropbox())
	add("Dropbox", filepath.Join(home, "Dropbox"))
	switch runtime.GOOS {
	case "darwin":
		add("iCloud Drive", filepath.Join(home, "Library", "Mobile Documents", "com~apple~CloudDocs"))
		if gd, _ := filepath.Glob(filepath.Join(home, "Library", "CloudStorage", "GoogleDrive-*", "My Drive")); len(gd) > 0 {
			add("Google Drive", gd[0])
		}
	case "windows":
		add("iCloud Drive", filepath.Join(home, "iCloudDrive"))
		for _, letter := range "GHIJ" {
			add("Google Drive", string(letter)+`:\My Drive`)
		}
	}
	return found
}

// dropbox is where Dropbox's own settings say its folder is.
func dropbox() string {
	for _, base := range []string{os.Getenv("APPDATA"), os.Getenv("LOCALAPPDATA"), filepath.Join(os.Getenv("HOME"), ".dropbox")} {
		data, err := os.ReadFile(filepath.Join(base, "Dropbox", "info.json"))
		if err != nil {
			data, err = os.ReadFile(filepath.Join(base, "info.json"))
		}
		if err != nil {
			continue
		}
		var info map[string]struct {
			Path string `json:"path"`
		}
		if json.Unmarshal(data, &info) == nil {
			if p := info["personal"].Path; p != "" {
				return p
			}
		}
	}
	return ""
}
