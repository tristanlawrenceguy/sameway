package soundtrack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func mp4box(typ string, parts ...[]byte) []byte {
	body := bytes.Join(parts, nil)
	b := binary.BigEndian.AppendUint32(nil, uint32(8+len(body)))
	return append(append(b, typ...), body...)
}

func u32(vs ...uint32) []byte {
	var b []byte
	for _, v := range vs {
		b = binary.BigEndian.AppendUint32(b, v)
	}
	return b
}

// An MP4's sound is found by its sample table and copied out as ADTS,
// frame by frame, in chunks at the times the table gives, the picture
// left behind.
func TestAnMP4sSoundIsCopiedOutAsADTS(t *testing.T) {
	frames := [][]byte{[]byte("frame-one"), []byte("frame-two!"), []byte("frame-3"), []byte("frame-four")}
	var media []byte
	for _, f := range frames {
		media = append(media, f...)
	}
	esds := append(u32(0), 0x03, 25, 0, 1, 0, 0x04, 17, 0x40, 0x15, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x05, 2, 0x12, 0x10)
	entry := mp4box("mp4a", make([]byte, 28), mp4box("esds", esds))
	var sizes []byte
	for _, f := range frames {
		sizes = binary.BigEndian.AppendUint32(sizes, uint32(len(f)))
	}
	build := func(offset uint32) []byte {
		stbl := mp4box("stbl",
			mp4box("stsd", u32(0, 1), entry),
			mp4box("stts", u32(0, 1, 4, 22050)), // half a second each at 44.1 kHz
			mp4box("stsc", u32(0, 1, 1, 4, 1)),
			mp4box("stsz", u32(0, 0, 4), sizes),
			mp4box("stco", u32(0, 1, offset)))
		trak := mp4box("trak", mp4box("mdia",
			mp4box("hdlr", u32(0, 0), []byte("soun"), make([]byte, 12)),
			mp4box("mdhd", u32(0, 0, 0, 44100, 0)),
			mp4box("minf", stbl)))
		video := mp4box("trak", mp4box("mdia", mp4box("hdlr", u32(0, 0), []byte("vide"), make([]byte, 12))))
		return mp4box("moov", video, trak)
	}
	ftyp := mp4box("ftyp", []byte("isom"), u32(0))
	moov := build(0)
	mdatAt := uint32(len(ftyp) + len(moov) + 8)
	file := append(append(append([]byte{}, ftyp...), build(mdatAt)...), mp4box("mdat", media)...)

	dir := t.TempDir()
	src := filepath.Join(dir, "talk.mp4")
	os.WriteFile(src, file, 0o644)
	Every = 1.0
	defer func() { Every = 600 }()
	chunks, err := Extract(src, filepath.Join(dir, "sound"))
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 || chunks[0].Start != 0 || chunks[1].Start != 1 || chunks[0].Ext != "aac" {
		t.Fatalf("two chunks of a second each, at 0 and 1: %+v", chunks)
	}
	first, _ := os.ReadFile(chunks[0].Path)
	// AAC-LC, 44.1 kHz, two channels; seven bytes of header and the frame.
	want := []byte{0xFF, 0xF1, 0x50, 0x80, byte((len(frames[0]) + 7) >> 3), byte(((len(frames[0])+7)&7)<<5 | 0x1F), 0xFC}
	if !bytes.HasPrefix(first, want) || !bytes.Contains(first, frames[1]) || bytes.Contains(first, frames[2]) {
		t.Errorf("the first chunk is the first two frames as ADTS: % x", first[:16])
	}
	if second, _ := os.ReadFile(chunks[1].Path); !bytes.Contains(second, frames[3]) {
		t.Error("the second chunk holds the rest")
	}
}

func ebmlEl(id uint32, body []byte) []byte {
	var idb []byte
	for s := 24; s >= 0; s -= 8 {
		if b := byte(id >> s); b != 0 || len(idb) > 0 {
			idb = append(idb, b)
		}
	}
	return append(append(idb, 0x01, 0, 0, 0, 0, 0, 0, byte(len(body))), body...)
}

// A WebM's sound track is found among its tracks, its blocks read from
// clusters of unknown size, and copied out as Ogg Opus that a decoder
// reads: its head first, each packet a page with a true checksum.
func TestAWebMsSoundIsCopiedOutAsOggOpus(t *testing.T) {
	head := append([]byte("OpusHead"), 1, 2, 0x38, 1, 0x80, 0xBB, 0, 0, 0, 0, 0)
	tracks := ebmlEl(idTracks, append(
		ebmlEl(idTrackEntry, append(append(ebmlEl(idTrackNumber, []byte{1}), ebmlEl(idTrackType, []byte{1})...), ebmlEl(idCodecID, []byte("V_VP8"))...)),
		ebmlEl(idTrackEntry, append(append(append(ebmlEl(idTrackNumber, []byte{2}), ebmlEl(idTrackType, []byte{2})...), ebmlEl(idCodecID, []byte("A_OPUS"))...), ebmlEl(idCodecPrivate, head)...))...))
	block := func(track byte, rel int16, data string) []byte {
		b := []byte{0x80 | track, byte(uint16(rel) >> 8), byte(rel), 0x80}
		return ebmlEl(idSimpleBlock, append(b, data...))
	}
	// A 20 ms CELT packet (config 31), and a 60 ms SILK one (config 3).
	cluster := append([]byte{0x1F, 0x43, 0xB6, 0x75, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, ebmlEl(idTimestamp, []byte{0})...)
	cluster = append(cluster, block(1, 0, "PICTURE")...)
	cluster = append(cluster, block(2, 0, "\xF8opus-a")...)
	cluster = append(cluster, block(2, 20, "\x18opus-b")...)
	file := append(ebmlEl(0x1A45DFA3, []byte{0x42, 0x82, 0x84, 'w', 'e', 'b', 'm'}), 0x18, 0x53, 0x80, 0x67, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF)
	file = append(append(file, tracks...), cluster...)

	dir := t.TempDir()
	src := filepath.Join(dir, "clip.webm")
	os.WriteFile(src, file, 0o644)
	chunks, err := Extract(src, filepath.Join(dir, "sound"))
	if err != nil || len(chunks) != 1 || chunks[0].Ext != "ogg" {
		t.Fatalf("one Ogg chunk: %+v %v", chunks, err)
	}
	out, _ := os.ReadFile(chunks[0].Path)
	if bytes.Contains(out, []byte("PICTURE")) {
		t.Error("the picture is left behind")
	}
	var pages [][]byte
	for p := out; len(p) >= 27; {
		if string(p[:4]) != "OggS" {
			t.Fatalf("pages one after another: % x", p[:8])
		}
		n := 27 + int(p[26])
		for _, l := range p[27:n] {
			n += int(l)
		}
		page := append([]byte{}, p[:n]...)
		sum := binary.LittleEndian.Uint32(page[22:])
		binary.LittleEndian.PutUint32(page[22:], 0)
		if oggUpdate(0, page) != sum {
			t.Errorf("page %d's checksum is Ogg's", len(pages))
		}
		pages = append(pages, p[:n])
		p = p[n:]
	}
	if len(pages) != 4 || !bytes.Contains(pages[0], []byte("OpusHead")) || pages[0][5] != 2 || !bytes.Contains(pages[1], []byte("OpusTags")) {
		t.Fatalf("head, tags, then a page a packet: %d pages", len(pages))
	}
	if g := binary.LittleEndian.Uint64(pages[2][6:]); g != 960 {
		t.Errorf("a 20 ms packet is 960 samples in, not %d", g)
	}
	if g := binary.LittleEndian.Uint64(pages[3][6:]); g != 960+2880 {
		t.Errorf("then a 60 ms one, not %d", g)
	}
}

// An MP3 is cut at its frames, past an ID3 tag, into chunks.
func TestAnMP3IsCutAtItsFrames(t *testing.T) {
	var file []byte
	file = append(file, 'I', 'D', '3', 3, 0, 0, 0, 0, 0, 10)
	file = append(file, make([]byte, 10)...)
	// MPEG 1 Layer III, 128 kbps, 44.1 kHz: 417 bytes, 1152 samples.
	for i := 0; i < 100; i++ {
		f := make([]byte, 417)
		copy(f, []byte{0xFF, 0xFB, 0x90, 0x00})
		f[4] = byte(i)
		file = append(file, f...)
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "talk.mp3")
	os.WriteFile(src, file, 0o644)
	Every = 1.0
	defer func() { Every = 600 }()
	chunks, err := Extract(src, filepath.Join(dir, "sound"))
	if err != nil {
		t.Fatal(err)
	}
	// 100 frames of 26 ms are 2.6 seconds: three chunks.
	if len(chunks) != 3 || chunks[0].Ext != "mp3" || chunks[1].Start < 0.99 || chunks[1].Start > 1.03 {
		t.Fatalf("three chunks at about 0, 1 and 2 seconds: %+v", chunks)
	}
	total := 0
	for _, c := range chunks {
		b, _ := os.ReadFile(c.Path)
		total += len(b)
		if len(b)%417 != 0 || b[0] != 0xFF {
			t.Error("each chunk is whole frames")
		}
	}
	if total != 417*100 {
		t.Errorf("every frame is copied once, got %d bytes", total)
	}
}

// Anything else is not copied out here, and says so.
func TestOtherFilesAreNotCopiedOut(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "notes.bin")
	os.WriteFile(src, []byte("just some bytes, not a recording"), 0o644)
	if _, err := Extract(src, filepath.Join(dir, "sound")); !errors.Is(err, ErrUnsupported) {
		t.Errorf("unsupported, not %v", err)
	}
}
