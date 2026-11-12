package media

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A gallery on a phone shows a few dozen photos, each twelve megapixels
// as the camera took it: tens of megabytes over the Wi-Fi for tiles a
// thumb wide. So each tile is the photo made small, once, and kept beside
// it (files/<id>.small.jpg); the file's own page still shows the original.
// The small one is turned the way the camera said it was held, since the
// original's EXIF, which the browser reads to turn it, is not carried
// over. A photo Go cannot read (HEIC, WebP) is given whole.

// smallSide is the longer side of a photo made small, in pixels: sharp
// in a tile on a phone's screen.
const smallSide = 480

// serveSmall is a picture made small for a gallery tile.
func (s *Service) serveSmall(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(records.FileType, r.PathValue("id"))
	if err != nil || rec.Fields["kind"] != "image" {
		http.NotFound(w, r)
		return
	}
	path, ok := s.StoredPath(rec)
	if !ok {
		http.NotFound(w, r)
		return
	}
	small := filepath.Join(s.app.Workspace.FilesDir(), rec.ID+".small.jpg")
	if _, err := os.Stat(small); err != nil && !makeSmall(path, small) {
		http.Redirect(w, r, "/files/"+rec.ID, http.StatusFound)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, small)
}

// makeSmall writes the picture at path made small, turned upright, to
// small; false when it cannot be read.
func makeSmall(path, small string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	head := make([]byte, convert.TakenHead)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false
	}
	// A file kept unseen, as from an email, can say it is too large to draw.
	if cfg, _, err := image.DecodeConfig(f); err == nil && tooLarge(cfg) {
		return false
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return false
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return false
	}
	out := upright(smaller(img, smallSide), convert.PhotoTurn(head))
	var b bytes.Buffer
	if err := jpeg.Encode(&b, out, &jpeg.Options{Quality: 80}); err != nil {
		return false
	}
	tmp := small + ".tmp"
	if err := os.WriteFile(tmp, b.Bytes(), 0o644); err != nil {
		return false
	}
	return os.Rename(tmp, small) == nil
}

// upright turns a picture the way EXIF's Orientation says the camera was
// held: 3 upside down, 6 a quarter turn clockwise, 8 anticlockwise; the
// mirrored ones (2, 4, 5, 7), which phones do not write, are left.
func upright(img image.Image, turn int) image.Image {
	if turn != 3 && turn != 6 && turn != 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	ow, oh := w, h
	if turn != 3 {
		ow, oh = h, w
	}
	out := image.NewRGBA(image.Rect(0, 0, ow, oh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(b.Min.X+x, b.Min.Y+y)
			switch turn {
			case 3:
				out.Set(w-1-x, h-1-y, c)
			case 6:
				out.Set(h-1-y, x, c)
			case 8:
				out.Set(y, w-1-x, c)
			}
		}
	}
	return out
}
