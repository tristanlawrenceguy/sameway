package server_test

import (
	"strings"
	"testing"
)

// Sameway's tab has its icon: every page names it, and it is served, the
// .ico a browser asks for by itself too.
func TestEveryPageHasSamewaysIcon(t *testing.T) {
	t.Parallel()
	_, h := newApp(t)
	if page := get(t, h, "/").Body.String(); !strings.Contains(page, `<link rel="icon" href="/favicon.svg" type="image/svg+xml">`) {
		t.Errorf("the page names its icon: %s", truncate(page))
	}
	svg := get(t, h, "/favicon.svg")
	if svg.Code != 200 || svg.Header().Get("Content-Type") != "image/svg+xml" || !strings.Contains(svg.Body.String(), "#1a45a8") {
		t.Errorf("the icon is served: %d %s", svg.Code, svg.Header().Get("Content-Type"))
	}
	if ico := get(t, h, "/favicon.ico"); ico.Code != 200 || !strings.HasPrefix(ico.Body.String(), "\x00\x00\x01\x00") {
		t.Errorf("and the .ico: %d", ico.Code)
	}
}
