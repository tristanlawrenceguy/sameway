package soundtrack

import (
	"encoding/binary"
	"errors"
	"io"
)

// MP4, MOV and M4A keep every frame of every track in one file, and say
// where in the moov box: the sound track's sample table gives each audio
// frame's offset, size and time, so the sound is copied out by reading only
// those bytes. A fragmented MP4 keeps its table in pieces (mp4frag.go).

// box is one ISO box: its type and what is inside it.
type box struct {
	typ  string
	body []byte
}

// boxes lists the boxes packed in b.
func boxes(b []byte) []box {
	var out []box
	for len(b) >= 8 {
		size := int(binary.BigEndian.Uint32(b))
		typ := string(b[4:8])
		head := 8
		switch size {
		case 0:
			size = len(b)
		case 1:
			if len(b) < 16 {
				return out
			}
			size, head = int(binary.BigEndian.Uint64(b[8:])), 16
		}
		if size < head || size > len(b) {
			return out
		}
		out = append(out, box{typ, b[head:size]})
		b = b[size:]
	}
	return out
}

func child(b []byte, path ...string) []byte {
	for _, want := range path {
		found := false
		for _, bx := range boxes(b) {
			if bx.typ == want {
				b, found = bx.body, true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return b
}

// topBox finds a top-level box in the file and reads it whole; the moov
// box is a table, megabytes at most, never the media.
func topBox(r io.ReaderAt, size int64, want string) ([]byte, error) {
	var off int64
	h := make([]byte, 16)
	for off+8 <= size {
		if _, err := r.ReadAt(h[:8], off); err != nil {
			return nil, err
		}
		n, head := int64(binary.BigEndian.Uint32(h)), int64(8)
		typ := string(h[4:8])
		if n == 1 {
			if _, err := r.ReadAt(h[8:16], off+8); err != nil {
				return nil, err
			}
			n, head = int64(binary.BigEndian.Uint64(h[8:16])), 16
		} else if n == 0 {
			n = size - off
		}
		if n < head || off+n > size {
			return nil, errors.New("the file's boxes are cut short")
		}
		if typ == want {
			if n-head > 256<<20 {
				return nil, ErrUnsupported
			}
			b := make([]byte, n-head)
			_, err := r.ReadAt(b, off+head)
			return b, err
		}
		off += n
	}
	return nil, ErrUnsupported
}

func readMP4(r io.ReaderAt, size int64) ([]frame, writer, error) {
	moov, err := topBox(r, size, "moov")
	if err != nil {
		return nil, nil, err
	}
	for _, t := range boxes(moov) {
		if t.typ != "trak" {
			continue
		}
		mdia := child(t.body, "mdia")
		if hdlr := child(mdia, "hdlr"); len(hdlr) < 12 || string(hdlr[8:12]) != "soun" {
			continue
		}
		scale := timescale(child(mdia, "mdhd"))
		stbl := child(mdia, "minf", "stbl")
		w, err := sampleEntry(child(stbl, "stsd"))
		if err != nil {
			return nil, nil, err
		}
		if child(moov, "mvex") != nil {
			frames, err := fragments(r, size, moov, trackID(t.body), scale)
			return frames, w, err
		}
		frames, err := sampleTable(stbl, scale)
		return frames, w, err
	}
	return nil, nil, errors.New("there is no sound in it")
}

func timescale(mdhd []byte) float64 {
	if len(mdhd) >= 24 && mdhd[0] == 1 {
		return float64(binary.BigEndian.Uint32(mdhd[20:]))
	}
	if len(mdhd) >= 16 {
		return float64(binary.BigEndian.Uint32(mdhd[12:]))
	}
	return 0
}

// sampleEntry reads what the sound is from the track's first sample
// description: AAC and MP3 (mp4a, with an esds saying which) or Opus.
func sampleEntry(stsd []byte) (writer, error) {
	if len(stsd) < 16 {
		return nil, ErrUnsupported
	}
	entry := boxes(stsd[8:])
	if len(entry) == 0 || len(entry[0].body) < 28 {
		return nil, ErrUnsupported
	}
	body := entry[0].body
	// The audio sample entry is 28 bytes; QuickTime's versions 1 and 2
	// add 16 and 36 more before the boxes inside it.
	inner := body[28:]
	switch binary.BigEndian.Uint16(body[8:]) {
	case 1:
		inner = body[min(len(body), 44):]
	case 2:
		inner = body[min(len(body), 64):]
	}
	switch entry[0].typ {
	case "mp4a":
		esds := findBox(inner, "esds")
		if esds == nil {
			return nil, ErrUnsupported
		}
		return fromESDS(esds)
	case ".mp3":
		return mp3Frames{}, nil
	case "Opus":
		dops := findBox(inner, "dOps")
		if len(dops) < 11 {
			return nil, ErrUnsupported
		}
		head := []byte("OpusHead")
		head = append(head, 1, dops[1])
		head = binary.LittleEndian.AppendUint16(head, binary.BigEndian.Uint16(dops[2:]))
		head = binary.LittleEndian.AppendUint32(head, binary.BigEndian.Uint32(dops[4:]))
		head = binary.LittleEndian.AppendUint16(head, binary.BigEndian.Uint16(dops[8:]))
		head = append(head, dops[10])
		if dops[10] != 0 {
			head = append(head, dops[11:]...)
		}
		return &oggOpus{head: head}, nil
	}
	return nil, ErrUnsupported
}

// findBox looks for a box anywhere inside b, as QuickTime nests esds in a
// wave box.
func findBox(b []byte, want string) []byte {
	for _, bx := range boxes(b) {
		if bx.typ == want {
			return bx.body
		}
		if bx.typ == "wave" {
			if f := findBox(bx.body, want); f != nil {
				return f
			}
		}
	}
	return nil
}

// fromESDS reads an MPEG-4 elementary stream descriptor: AAC with its
// configuration, or MP3.
func fromESDS(b []byte) (writer, error) {
	if len(b) < 4 {
		return nil, ErrUnsupported
	}
	b = b[4:]
	desc := func(b []byte) (byte, []byte, []byte) {
		if len(b) < 2 {
			return 0, nil, nil
		}
		tag, i, n := b[0], 1, 0
		for ; i < len(b) && i < 5; i++ {
			n = n<<7 | int(b[i]&0x7F)
			if b[i]&0x80 == 0 {
				i++
				break
			}
		}
		if i+n > len(b) {
			n = len(b) - i
		}
		return tag, b[i : i+n], b[i+n:]
	}
	tag, es, _ := desc(b)
	if tag != 3 || len(es) < 3 {
		return nil, ErrUnsupported
	}
	flags, es := es[2], es[3:]
	if flags&0x80 != 0 {
		es = es[min(len(es), 2):]
	}
	if flags&0x40 != 0 && len(es) > 0 {
		es = es[min(len(es), 1+int(es[0])):]
	}
	if flags&0x20 != 0 {
		es = es[min(len(es), 2):]
	}
	tag, dc, _ := desc(es)
	if tag != 4 || len(dc) < 13 {
		return nil, ErrUnsupported
	}
	switch dc[0] {
	case 0x6B, 0x69:
		return mp3Frames{}, nil
	case 0x40, 0x66, 0x67, 0x68:
		tag, asc, _ := desc(dc[13:])
		if tag != 5 {
			return nil, ErrUnsupported
		}
		return parseASC(asc)
	}
	return nil, ErrUnsupported
}
