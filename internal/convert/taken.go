package convert

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"time"
)

// When a photo or video was taken, as the file itself says: a phone's
// photo shared today may be from last summer, and the day it was added
// says nothing of it. A photo (JPEG, HEIC, PNG, WebP) says it in its
// EXIF: DateTimeOriginal, with OffsetTimeOriginal when the phone wrote
// one, else the camera's clock as it read where it was. A video (MP4,
// MOV) says it in its movie header, in UTC.

// TakenHead is how much of a photo is read for when it was taken: its
// EXIF comes first, in at most 64 KB.
const TakenHead = 256 << 10

// PhotoTaken is when a photo was taken, from the start of its file. With
// zoned false it is the camera's clock, to be read in the person's zone.
func PhotoTaken(head []byte) (at time.Time, zoned, ok bool) {
	tiff := exifTIFF(head)
	if tiff == nil {
		return time.Time{}, false, false
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return time.Time{}, false, false
	}
	ifd0 := tags(tiff, order, int(order.Uint32(tiff[4:8])))
	stamp, offset := ifd0[0x0132], ""
	if sub, ok := ifd0[0x8769]; ok && len(sub) == 4 {
		exif := tags(tiff, order, int(order.Uint32(sub)))
		if v := exif[0x9003]; len(v) > 0 {
			stamp, offset = v, string(bytes.TrimRight(exif[0x9011], "\x00 "))
		}
	}
	s := string(bytes.TrimRight(stamp, "\x00 "))
	if offset != "" {
		if t, err := time.Parse("2006:01:02 15:04:05-07:00", s+offset); err == nil {
			return t, true, t.Year() > 1900
		}
	}
	t, err := time.Parse("2006:01:02 15:04:05", s)
	return t, false, err == nil && t.Year() > 1900
}

// exifTIFF finds the TIFF block EXIF is written in: after "Exif\0\0" in a
// JPEG's APP1, a HEIC's Exif item or a WebP's EXIF chunk, or a PNG's eXIf
// chunk, which holds it bare.
func exifTIFF(head []byte) []byte {
	if i := bytes.Index(head, []byte("Exif\x00\x00")); i >= 0 && len(head) > i+14 {
		return head[i+6:]
	}
	if i := bytes.Index(head, []byte("eXIf")); i >= 0 && len(head) > i+12 {
		return head[i+4:]
	}
	return nil
}

// tags reads one directory of TIFF tags: each tag's value, the bytes it
// holds in place or where it points.
func tags(tiff []byte, order binary.ByteOrder, at int) map[uint16][]byte {
	out := map[uint16][]byte{}
	if at < 8 || at+2 > len(tiff) {
		return out
	}
	n := int(order.Uint16(tiff[at:]))
	for i := 0; i < n; i++ {
		e := at + 2 + 12*i
		if e+12 > len(tiff) {
			break
		}
		typ, count := order.Uint16(tiff[e+2:]), int(order.Uint32(tiff[e+4:]))
		size := count
		if typ == 4 { // a LONG, as the pointer to the EXIF tags is
			size = 4 * count
		}
		if size <= 4 {
			out[order.Uint16(tiff[e:])] = tiff[e+8 : e+8+size]
			continue
		}
		off := int(order.Uint32(tiff[e+8:]))
		if off > 0 && size < 256 && off+size <= len(tiff) {
			out[order.Uint16(tiff[e:])] = tiff[off : off+size]
		}
	}
	return out
}

// VideoTaken is when a video was made, from its movie header, which a
// phone may write at the end of the file.
func VideoTaken(r io.ReaderAt, size int64) (time.Time, bool) {
	moov := box(r, 0, size, "moov")
	if moov == nil {
		return time.Time{}, false
	}
	i := bytes.Index(moov, []byte("mvhd"))
	if i < 0 || len(moov) < i+12 {
		return time.Time{}, false
	}
	var secs uint64
	if moov[i+4] == 1 {
		if len(moov) < i+16 {
			return time.Time{}, false
		}
		secs = binary.BigEndian.Uint64(moov[i+8:])
	} else {
		secs = uint64(binary.BigEndian.Uint32(moov[i+8:]))
	}
	if secs == 0 {
		return time.Time{}, false
	}
	t := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(secs) * time.Second)
	return t, t.Year() > 1970 && t.Year() < 2200
}

// box is the contents of the first top-level box of a kind, up to 16 MB.
func box(r io.ReaderAt, at, end int64, kind string) []byte {
	head := make([]byte, 16)
	for at+8 <= end {
		if _, err := r.ReadAt(head[:8], at); err != nil {
			return nil
		}
		n, name, skip := int64(binary.BigEndian.Uint32(head)), string(head[4:8]), int64(8)
		switch n {
		case 1:
			if _, err := r.ReadAt(head[8:16], at+8); err != nil {
				return nil
			}
			n, skip = int64(binary.BigEndian.Uint64(head[8:16])), 16
		case 0:
			n = end - at
		}
		if n < skip || strings.ContainsAny(name, "\x00") {
			return nil
		}
		if name == kind {
			body := make([]byte, min(n-skip, 16<<20))
			if _, err := r.ReadAt(body, at+skip); err != nil && err != io.EOF {
				return nil
			}
			return body
		}
		at += n
	}
	return nil
}
