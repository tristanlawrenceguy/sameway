package server

import (
	"bytes"
	"image"
	"image/gif"
	_ "image/jpeg" // read the size of a JPEG
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A picture on its file's page: its size known before it loads, so the
// page does not jump; loaded at once, since it is the first thing there;
// and, when it moves, still until someone asks it to play (WCAG 2.2.2).

// pictureOf is the image component's props for a file that is a picture.
func (s *Server) pictureOf(rec *store.Record, alt string) map[string]any {
	props := map[string]any{"src": "/files/" + rec.ID, "alt": alt, "loading": "eager"}
	path, ok := s.storedPath(rec)
	if !ok {
		return props
	}
	f, err := os.Open(path)
	if err != nil {
		return props
	}
	defer f.Close()
	// Only the header is read: the size, not the picture.
	if cfg, _, err := image.DecodeConfig(f); err == nil && cfg.Width > 0 && cfg.Height > 0 {
		props["width"], props["height"] = cfg.Width, cfg.Height
	}
	if strings.EqualFold(filepath.Ext(path), ".gif") && moving(path) {
		props["still"] = "/files/" + rec.ID + "/still"
	}
	return props
}

// storedPath is where a file's original is kept, when it is.
func (s *Server) storedPath(rec *store.Record) (string, bool) {
	stored, _ := rec.Fields["path"].(string)
	if stored == "" || strings.ContainsAny(stored, `/\`) {
		return "", false
	}
	return filepath.Join(s.app.Workspace.FilesDir(), stored), true
}

// moving is whether a GIF has more than one frame.
func moving(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	g, err := gif.DecodeAll(bytes.NewReader(data))
	return err == nil && len(g.Image) > 1
}

// serveStill is a moving GIF's first frame, as a PNG: the picture at rest.
func (s *Server) serveStill(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(FileType, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	path, ok := s.storedPath(rec)
	if !ok {
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil || len(g.Image) == 0 {
		http.NotFound(w, r)
		return
	}
	var b bytes.Buffer
	if err := png.Encode(&b, g.Image[0]); err != nil {
		http.Error(w, "could not make the still picture", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(b.Bytes())
}

// fileSafety keeps an uploaded file from acting as this site when it is
// opened on its own: an SVG or an HTML page can carry a script, and served
// from here it would run as the workspace. The browser takes the type as
// given and runs nothing in it. A PDF is left to the browser's own viewer,
// which the sandbox would stop.
func fileSafety(w http.ResponseWriter, contentType string) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !strings.HasPrefix(contentType, "application/pdf") {
		w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'; media-src 'self'")
	}
}
