package soundtrack

import (
	"bufio"
	"io"
)

// An MP3 file is its frames, each with a four-byte header that says how
// long it is: read one after another, skipping an ID3 tag at the start
// and anything that is not a frame.

var mp3Rates = [4][4]int{ // [version][index]; version 0 is MPEG 2.5, 2 is MPEG 2, 3 is MPEG 1
	{11025, 12000, 8000, 0}, {0, 0, 0, 0}, {22050, 24000, 16000, 0}, {44100, 48000, 32000, 0},
}
var mp3Kbps = [2][16]int{ // Layer III: MPEG 1, then MPEG 2 and 2.5
	{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0},
	{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},
}

// mp3Header reads a Layer III frame header: its length in bytes and how
// many samples it holds at what rate; ok is false for anything else.
func mp3Header(h []byte) (length, samples, rate int, ok bool) {
	if len(h) < 4 || h[0] != 0xFF || h[1]&0xE0 != 0xE0 {
		return 0, 0, 0, false
	}
	version, layer := int(h[1]>>3)&3, int(h[1]>>1)&3
	if version == 1 || layer != 1 {
		return 0, 0, 0, false
	}
	br, ri, pad := int(h[2]>>4), int(h[2]>>2)&3, int(h[2]>>1)&1
	rate = mp3Rates[version][ri]
	table := 1
	if version == 3 {
		table = 0
	}
	kbps := mp3Kbps[table][br]
	if rate == 0 || kbps == 0 {
		return 0, 0, 0, false
	}
	if version == 3 {
		return 144*kbps*1000/rate + pad, 1152, rate, true
	}
	return 72*kbps*1000/rate + pad, 576, rate, true
}

func readMP3(r io.ReaderAt, size int64) ([]frame, writer, error) {
	br := bufio.NewReaderSize(io.NewSectionReader(r, 0, size), 1<<16)
	var off int64
	skip := func(n int64) {
		br.Discard(int(n))
		off += n
	}
	if h, err := br.Peek(10); err == nil && string(h[:3]) == "ID3" {
		skip(10 + (int64(h[6]&0x7F)<<21 | int64(h[7]&0x7F)<<14 | int64(h[8]&0x7F)<<7 | int64(h[9]&0x7F)))
	}
	var frames []frame
	var at float64
	for off < size {
		h, err := br.Peek(4)
		if err != nil {
			break
		}
		n, samples, rate, ok := mp3Header(h)
		if !ok || off+int64(n) > size {
			skip(1)
			continue
		}
		frames = append(frames, frame{off: off, size: n, at: at})
		at += float64(samples) / float64(rate)
		skip(int64(n))
	}
	if len(frames) == 0 {
		return nil, nil, ErrUnsupported
	}
	return frames, mp3Frames{}, nil
}
