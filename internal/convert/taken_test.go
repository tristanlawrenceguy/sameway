package convert

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

// exifBlock is a TIFF block as a phone writes it: IFD0 pointing to the
// EXIF tags, which say when it was taken and, when given, its offset.
func exifBlock(stamp, offset string) []byte {
	le := binary.LittleEndian
	var b bytes.Buffer
	b.WriteString("II*\x00")
	binary.Write(&b, le, uint32(8))
	// IFD0 at 8: one tag, the EXIF pointer, to 26.
	binary.Write(&b, le, uint16(1))
	binary.Write(&b, le, []uint16{0x8769, 4})
	binary.Write(&b, le, []uint32{1, 26})
	binary.Write(&b, le, uint32(0))
	// EXIF tags at 26: when, and its offset, their words after them at 56.
	n := uint16(1)
	if offset != "" {
		n = 2
	}
	binary.Write(&b, le, n)
	at := uint32(26 + 2 + 12*int(n) + 4)
	binary.Write(&b, le, []uint16{0x9003, 2})
	binary.Write(&b, le, []uint32{uint32(len(stamp) + 1), at})
	if offset != "" {
		binary.Write(&b, le, []uint16{0x9011, 2})
		binary.Write(&b, le, []uint32{uint32(len(offset) + 1), at + uint32(len(stamp)+1)})
	}
	binary.Write(&b, le, uint32(0))
	b.WriteString(stamp + "\x00")
	if offset != "" {
		b.WriteString(offset + "\x00")
	}
	return b.Bytes()
}

// A photo says when it was taken: a JPEG's EXIF with the offset the phone
// gave, a PNG's with none (the camera's clock, read where it was); one
// that says nothing, or says nothing real, is not given a day.
func TestAPhotoSaysWhenItWasTaken(t *testing.T) {
	t.Parallel()
	jpeg := append([]byte("\xff\xd8\xff\xe1\x00\x60Exif\x00\x00"), exifBlock("2026:07:14 16:32:05", "+01:00")...)
	at, zoned, ok := PhotoTaken(jpeg)
	if !ok || !zoned || !at.Equal(time.Date(2026, 7, 14, 15, 32, 5, 0, time.UTC)) {
		t.Errorf("a JPEG with its offset: %v %v %v", at, zoned, ok)
	}
	png := append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x40eXIf"), exifBlock("2025:12:25 09:00:00", "")...)
	at, zoned, ok = PhotoTaken(png)
	if !ok || zoned || at.Format("2006-01-02 15:04") != "2025-12-25 09:00" {
		t.Errorf("a PNG with the camera's clock: %v %v %v", at, zoned, ok)
	}
	for _, none := range [][]byte{[]byte("\xff\xd8\xff\xe0JFIF"), append([]byte("Exif\x00\x00"), exifBlock("0000:00:00 00:00:00", "")...), []byte("Exif\x00\x00II*\x00\xff\xff\xff\xff")} {
		if _, _, ok := PhotoTaken(none); ok {
			t.Errorf("no day from %q", none)
		}
	}
}

// A video says when it was made in its movie header, which a phone may
// write after the pictures.
func TestAVideoSaysWhenItWasMade(t *testing.T) {
	t.Parallel()
	made := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
	secs := uint32(made.Sub(time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)) / time.Second)
	boxOf := func(kind string, body []byte) []byte {
		b := binary.BigEndian.AppendUint32(nil, uint32(8+len(body)))
		return append(append(b, kind...), body...)
	}
	mvhd := boxOf("mvhd", binary.BigEndian.AppendUint32(make([]byte, 4), secs))
	file := append(append(boxOf("ftyp", []byte("isom")), boxOf("mdat", make([]byte, 1000))...), boxOf("moov", mvhd)...)
	at, ok := VideoTaken(bytes.NewReader(file), int64(len(file)))
	if !ok || !at.Equal(made) {
		t.Errorf("made %v: %v %v", made, at, ok)
	}
	if _, ok := VideoTaken(bytes.NewReader(file[:20]), 20); ok {
		t.Errorf("no header, no day")
	}
}
