package server_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Sameway can be installed as an app: every page names a manifest with its
// icons, the manifest and icons are served, and Help offers Install where
// the browser can (38-install.js) and says the other ways.
func TestSamewayInstallsAsAnApp(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	page := get(t, h, "/").Body.String()
	for _, want := range []string{`<link rel="manifest" href="/manifest.webmanifest">`, `<link rel="apple-touch-icon" href="/icon-square-180.png">`} {
		if !strings.Contains(page, want) {
			t.Errorf("the page names %s", want)
		}
	}
	var m struct {
		Name    string `json:"name"`
		Display string `json:"display"`
		Icons   []struct {
			Src string `json:"src"`
		} `json:"icons"`
	}
	if err := json.Unmarshal(get(t, h, "/manifest.webmanifest").Body.Bytes(), &m); err != nil || m.Name != "Sameway" || m.Display != "standalone" || len(m.Icons) < 2 {
		t.Fatalf("the manifest: %v %+v", err, m)
	}
	for _, i := range m.Icons {
		if res := get(t, h, i.Src); res.Code != 200 || res.Header().Get("Content-Type") != "image/png" {
			t.Errorf("%s is served: %d", i.Src, res.Code)
		}
	}
	help := get(t, h, "/help").Body.String()
	if !strings.Contains(help, "data-install hidden") || !strings.Contains(help, "Add to Home Screen") {
		t.Errorf("Help offers it: %s", truncate(help))
	}
	js, _ := os.ReadFile("../../design/base/38-install.js")
	if !strings.Contains(string(js), "beforeinstallprompt") || !strings.Contains(string(js), "asked.prompt()") {
		t.Error("the button asks the browser to install")
	}
}
