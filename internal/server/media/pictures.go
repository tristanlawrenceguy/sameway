package media

import (
	"bytes"
	"image"
	"image/color"
	_ "image/gif" // a GIF's first frame
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/llm"
	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// A picture goes to a model that can see as it is when it is small, and
// otherwise made smaller first: no longer than 1568 pixels on its long
// side, which is as much as the models look at, as a JPEG. SVG is left
// out (a drawing's words are its text); WebP and AVIF go as they are when
// small enough, since only the standard library's formats are redrawn.

const (
	pictureSide  = 1568
	pictureBytes = 3 << 20
)

var pictureTypes = map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp"}

// PictureFor is a picture file ready for a model to see.
func (s *Service) PictureFor(id string) (llm.Image, bool) {
	rec, err := s.app.Store.Get(records.FileType, id)
	if err != nil || rec.Fields["kind"] != "image" {
		return llm.Image{}, false
	}
	path, ok := s.StoredPath(rec)
	if !ok {
		return llm.Image{}, false
	}
	typ := pictureTypes[strings.ToLower(filepath.Ext(path))]
	if typ == "" {
		return llm.Image{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return llm.Image{}, false
	}
	cfg, _, cerr := image.DecodeConfig(bytes.NewReader(data))
	if len(data) <= pictureBytes && (cerr != nil || max(cfg.Width, cfg.Height) <= pictureSide) {
		return llm.Image{Type: typ, Data: data}, true
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return llm.Image{}, false // too big, and in a form not redrawn here
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, smaller(src, pictureSide), &jpeg.Options{Quality: 85}); err != nil {
		return llm.Image{}, false
	}
	return llm.Image{Type: "image/jpeg", Data: out.Bytes()}, true
}

// smaller draws a picture no longer than side on its long side, each new
// pixel the average of the ones it covers.
func smaller(src image.Image, side int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if max(w, h) <= side {
		return src
	}
	nw, nh := w*side/max(w, h), h*side/max(w, h)
	dst := image.NewRGBA(image.Rect(0, 0, max(nw, 1), max(nh, 1)))
	for y := 0; y < nh; y++ {
		y0, y1 := b.Min.Y+y*h/nh, b.Min.Y+(y+1)*h/nh
		for x := 0; x < nw; x++ {
			x0, x1 := b.Min.X+x*w/nw, b.Min.X+(x+1)*w/nw
			var r, g, bl, n uint64
			for yy := y0; yy < max(y1, y0+1); yy++ {
				for xx := x0; xx < max(x1, x0+1); xx++ {
					cr, cg, cb, _ := src.At(xx, yy).RGBA()
					r, g, bl, n = r+uint64(cr), g+uint64(cg), bl+uint64(cb), n+1
				}
			}
			dst.Set(x, y, color.RGBA{uint8(r / n >> 8), uint8(g / n >> 8), uint8(bl / n >> 8), 255})
		}
	}
	return dst
}
