package server_test

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/server"
)

// A file opened on its own cannot act as the site: an SVG or HTML page
// carrying a script runs nothing, served from here.
func TestAnUploadedFileRunsNothingWhenOpened(t *testing.T) {
	a, h := newApp(t)
	dir := a.Workspace.FilesDir()
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "x.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), 0o644)
	rec, _ := a.Store.Create(server.FileType, map[string]any{"title": "x", "name": "x.svg", "path": "x.svg", "kind": "image"})
	res := get(t, h, "/files/"+rec.ID)
	if csp := res.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") {
		t.Errorf("an uploaded file is served sandboxed, got %q", csp)
	}
	if res.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("an uploaded file is served as the type it is, not sniffed")
	}
}

// A picture's size is known before it loads, so the page does not jump;
// one that moves starts still, its first frame served as a picture.
func TestAPictureKeepsItsPlaceAndAMovingOneStartsStill(t *testing.T) {
	a, h := newApp(t)
	dir := a.Workspace.FilesDir()
	os.MkdirAll(dir, 0o755)
	pal := color.Palette{color.White, color.Black}
	frame := func(c uint8) *image.Paletted {
		p := image.NewPaletted(image.Rect(0, 0, 40, 30), pal)
		p.Pix[0] = c
		return p
	}
	var b bytes.Buffer
	gif.EncodeAll(&b, &gif.GIF{Image: []*image.Paletted{frame(0), frame(1)}, Delay: []int{10, 10}})
	os.WriteFile(filepath.Join(dir, "wave.gif"), b.Bytes(), 0o644)
	rec, _ := a.Store.Create(server.FileType, map[string]any{"title": "Wave", "name": "wave.gif", "path": "wave.gif", "kind": "image", "description": "A hand waving"})

	page := get(t, h, "/t/file/"+rec.ID).Body.String()
	for _, want := range []string{`width="40" height="30"`, `src="/files/` + rec.ID + `/still"`, `data-moving="/files/` + rec.ID + `"`, `loading="eager"`} {
		if !strings.Contains(page, want) {
			t.Errorf("the picture should carry %s\n%s", want, truncate(page))
		}
	}
	still := get(t, h, "/files/"+rec.ID+"/still")
	if still.Code != 200 || still.Header().Get("Content-Type") != "image/png" {
		t.Errorf("the still is the first frame as a PNG, got %d %q", still.Code, still.Header().Get("Content-Type"))
	}
}
