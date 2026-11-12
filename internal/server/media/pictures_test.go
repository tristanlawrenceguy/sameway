package media

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
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

// A phone photo is kept as the sensor took it, with EXIF saying how the
// phone was held; made smaller, it is turned that way, so a letter
// photographed upright does not reach the model on its side.
func TestAPhotoMadeSmallerIsTurnedUpright(t *testing.T) {
	t.Parallel()
	stored := image.NewRGBA(image.Rect(0, 0, 2000, 1000))
	for y := 0; y < 1000; y++ {
		for x := 0; x < 2000; x++ {
			c := color.RGBA{255, 255, 255, 255}
			if x < 500 { // the top of the page, as the sensor kept it
				c = color.RGBA{255, 0, 0, 255}
			}
			stored.Set(x, y, c)
		}
	}
	var b bytes.Buffer
	jpeg.Encode(&b, stored, nil)
	// An APP1 segment whose one tag, Orientation, is 6: turn it a
	// quarter clockwise to show.
	exif := []byte("\xff\xe1\x00\x22Exif\x00\x00II*\x00\x08\x00\x00\x00\x01\x00\x12\x01\x03\x00\x01\x00\x00\x00\x06\x00\x00\x00\x00\x00\x00\x00")
	data := append(append([]byte("\xff\xd8"), exif...), b.Bytes()[2:]...)
	img, ok := fitted("image/jpeg", data)
	if !ok {
		t.Fatal("the photo is made smaller")
	}
	got, err := jpeg.Decode(bytes.NewReader(img.Data))
	if err != nil {
		t.Fatal(err)
	}
	if s := got.Bounds().Size(); s.X != 784 || s.Y != 1568 {
		t.Fatalf("turned upright, it stands: %v", s)
	}
	top, bottom := got.At(392, 20), got.At(392, 1540)
	if r, g, _, _ := top.RGBA(); r>>8 < 200 || g>>8 > 80 {
		t.Errorf("the top of the page is at the top: %v", top)
	}
	if _, g, _, _ := bottom.RGBA(); g>>8 < 200 {
		t.Errorf("and the rest below it: %v", bottom)
	}
}
