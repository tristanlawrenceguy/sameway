package apps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Connecting an app has to be as easy to take back as it was to give
// (GDPR Art. 7(3)), so the page reads which workspace an app's sameway
// entry starts, and takes the entry out again leaving every other server
// in the file as it was.

// Entry is the workspace the sameway entry in an app's file starts (the
// file for this workspace, for an app that keeps one in it), "" when the
// file has none. An entry that names no workspace is "?".
func Entry(app App, workspace string) string {
	raw, err := os.ReadFile(app.Path(workspace))
	if err != nil {
		return ""
	}
	var args []string
	if app.Shape == "toml" {
		section, ok := tomlSection(string(raw), "["+app.Key+".sameway]")
		if !ok {
			return ""
		}
		for _, line := range section {
			if k, v, ok := strings.Cut(line, "="); ok && strings.TrimSpace(k) == "args" {
				json.Unmarshal([]byte(strings.TrimSpace(v)), &args)
			}
		}
	} else {
		// The rest of the file is whatever the app keeps; only its servers
		// are read.
		var doc map[string]json.RawMessage
		var servers map[string]struct {
			Args []string `json:"args"`
		}
		if json.Unmarshal(raw, &doc) != nil || json.Unmarshal(doc[app.Key], &servers) != nil {
			return ""
		}
		server, ok := servers["sameway"]
		if !ok {
			return ""
		}
		args = server.Args
	}
	for i, a := range args {
		if a == "--workspace" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return "?"
}

// Same says whether two workspace paths are the same folder.
func Same(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// Remove takes the sameway entry out of the app's file, keeping every
// other server there. A file without one is left alone.
func Remove(path string, app App) error {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if app.Shape == "toml" {
		text, changed := withoutSection(string(raw), "["+app.Key+".sameway]")
		if !changed {
			return nil
		}
		return os.WriteFile(path, []byte(text), 0o644)
	}
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	servers, _ := doc[app.Key].(map[string]any)
	if _, ok := servers["sameway"]; !ok {
		return nil
	}
	delete(servers, "sameway")
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

// tomlSection is the lines under a section's header, up to the next one.
func tomlSection(text, head string) ([]string, bool) {
	var out []string
	in, found := false, false
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") {
			in = t == head
			found = found || in
			continue
		}
		if in {
			out = append(out, t)
		}
	}
	return out, found
}

// withoutSection is the text with a section and its lines taken out.
func withoutSection(text, head string) (string, bool) {
	var out []string
	in, changed := false, false
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "[") {
			in = t == head
			changed = changed || in
		}
		if !in {
			out = append(out, line)
		}
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n", changed
}
