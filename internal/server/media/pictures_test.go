package media

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"testing"
)

// A small PNG that says it is 100,000 pixels square is not drawn: read
// unattended from an email, it would take gigabytes.
func TestAPictureTooLargeToDrawIsNotDrawn(t *testing.T) {
	t.Parallel()
	var b bytes.Buffer
	png.Encode(&b, image.NewGray(image.Rect(0, 0, 1, 1)))
	data := b.Bytes()
	ihdr := data[8+8 : 8+8+13] // after the signature, the IHDR chunk's length and name
	binary.BigEndian.PutUint32(ihdr[0:], 100000)
	binary.BigEndian.PutUint32(ihdr[4:], 100000)
	binary.BigEndian.PutUint32(data[8+8+13:], crc32.ChecksumIEEE(data[8+4:8+8+13]))
	if cfg, err := png.DecodeConfig(bytes.NewReader(data)); err != nil || cfg.Width != 100000 {
		t.Fatalf("the crafted header reads as 100,000 pixels wide: %v %+v", err, cfg)
	}
	if img, ok := fitted("image/png", data); ok {
		t.Errorf("a picture too large to draw is unreadable, got %d bytes", len(img.Data))
	}
}
