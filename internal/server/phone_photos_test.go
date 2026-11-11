package server_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// jpegHeld is a JPEG as a phone writes one held on its side: the picture
// as the sensor saw it, w by h, and EXIF saying how to turn it.
func jpegHeld(t *testing.T, w, h int, turn uint16) []byte {
	t.Helper()
	var pic bytes.Buffer
	if err := jpeg.Encode(&pic, image.NewRGBA(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatal(err)
	}
	le := binary.LittleEndian
	var tiff bytes.Buffer
	tiff.WriteString("II*\x00")
	binary.Write(&tiff, le, uint32(8))
	binary.Write(&tiff, le, uint16(1))
	binary.Write(&tiff, le, []uint16{0x0112, 3})
	binary.Write(&tiff, le, uint32(1))
	binary.Write(&tiff, le, []uint16{turn, 0})
	binary.Write(&tiff, le, uint32(0))
	app1 := append([]byte("Exif\x00\x00"), tiff.Bytes()...)
	var out bytes.Buffer
	out.Write(pic.Bytes()[:2])
	out.Write([]byte{0xff, 0xe1})
	binary.Write(&out, binary.BigEndian, uint16(len(app1)+2))
	out.Write(app1)
	out.Write(pic.Bytes()[2:])
	return out.Bytes()
}

// From a phone: Save to Sameway takes photos, several at once, which on
// a phone offers the camera and the library, and the installed app's
// icon leads there when held. A gallery shows each photo made small, not
// whole, turned the way the phone was held.
func TestPhotosFromAPhoneAreSavedAndShownSmall(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	form := get(t, h, "/share").Body.String()
	if !strings.Contains(form, `type="file" name="files" multiple`) {
		t.Errorf("Save to Sameway takes several photos: %.2000s", form)
	}
	if m := get(t, h, "/manifest.webmanifest").Body.String(); !strings.Contains(m, `"shortcuts":[{"name":"Save to Sameway"`) || !strings.Contains(m, `"url":"/share"`) {
		t.Errorf("the app's icon, held, leads there: %s", m)
	}

	small := func(name string, data []byte) image.Config {
		t.Helper()
		body, ct := multipartFile(t, name, string(data), nil)
		id := strings.TrimPrefix(do(t, h, "POST", "/t/file/upload", body, ct).Header().Get("Location"), "/t/file/")
		if f, err := a.Store.Get(records.FileType, id); err != nil || f.Fields["kind"] != "image" {
			t.Fatalf("%s kept as a picture: %v", name, err)
		}
		res := get(t, h, "/files/"+id+"/small")
		if res.Code != 200 || res.Header().Get("Content-Type") != "image/jpeg" {
			t.Fatalf("%s made small: %d %s", name, res.Code, res.Header().Get("Content-Type"))
		}
		cfg, err := jpeg.DecodeConfig(res.Body)
		if err != nil {
			t.Fatalf("%s made small: %v", name, err)
		}
		again := get(t, h, "/files/"+id+"/small")
		if again.Code != 200 {
			t.Errorf("%s made small is kept: %d", name, again.Code)
		}
		return cfg
	}
	if cfg := small("Screenshot.png", pngOf(t, 1200, 600)); cfg.Width != 480 || cfg.Height != 240 {
		t.Errorf("a large picture is made no longer than 480: %dx%d", cfg.Width, cfg.Height)
	}
	if cfg := small("IMG_0001.jpg", jpegHeld(t, 60, 20, 6)); cfg.Width != 20 || cfg.Height != 60 {
		t.Errorf("a photo taken upright stands upright: %dx%d", cfg.Width, cfg.Height)
	}
}
