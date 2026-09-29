package soundtrack

import (
	"encoding/binary"
	"errors"
	"io"
)

// AAC is written as ADTS: each frame with a seven-byte header that says
// what it is, so the stream decodes from any frame with nothing before it.
type adts struct{ profile, freq, channels int }

var aacRates = []int{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000, 7350}

// parseASC reads an AAC AudioSpecificConfig, as MP4 and Matroska keep it.
func parseASC(b []byte) (*adts, error) {
	if len(b) < 2 {
		return nil, errors.New("the AAC sound has no configuration")
	}
	obj := int(b[0] >> 3)
	freq := int(b[0]&7)<<1 | int(b[1]>>7)
	ch := int(b[1]>>3) & 0xF
	if obj == 31 || freq == 15 {
		return nil, ErrUnsupported
	}
	// HE-AAC is carried as its AAC core, which every decoder plays.
	if obj == 5 || obj == 29 || obj > 4 || obj == 0 {
		obj = 2
	}
	if ch == 0 {
		ch = 2
	}
	return &adts{profile: obj - 1, freq: freq, channels: ch}, nil
}

func (a *adts) rate() int {
	if a.freq < len(aacRates) {
		return aacRates[a.freq]
	}
	return 44100
}

func (a *adts) begin(io.Writer) error { return nil }
func (a *adts) end(io.Writer) error   { return nil }
func (a *adts) ext() string           { return "aac" }
func (a *adts) frame(w io.Writer, data []byte, _ float64) error {
	n := len(data) + 7
	h := [7]byte{0xFF, 0xF1,
		byte(a.profile<<6 | a.freq<<2 | a.channels>>2),
		byte((a.channels&3)<<6 | n>>11),
		byte(n >> 3),
		byte((n&7)<<5 | 0x1F),
		0xFC}
	if _, err := w.Write(h[:]); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// MP3 frames are whole in themselves: a chunk is its frames, one after
// another.
type mp3Frames struct{}

func (mp3Frames) begin(io.Writer) error { return nil }
func (mp3Frames) end(io.Writer) error   { return nil }
func (mp3Frames) ext() string           { return "mp3" }
func (mp3Frames) frame(w io.Writer, data []byte, _ float64) error {
	_, err := w.Write(data)
	return err
}

// Opus is written as Ogg Opus: its head, its tags, then a page a packet.
type oggOpus struct {
	head    []byte
	seq     uint32
	granule int64
}

func (o *oggOpus) ext() string { return "ogg" }

func (o *oggOpus) begin(w io.Writer) error {
	o.seq, o.granule = 0, 0
	if err := o.page(w, o.head, 0x02, 0); err != nil {
		return err
	}
	tags := append([]byte("OpusTags"), 7, 0, 0, 0)
	tags = append(tags, "sameway"...)
	tags = append(tags, 0, 0, 0, 0)
	return o.page(w, tags, 0, 0)
}

func (o *oggOpus) frame(w io.Writer, data []byte, _ float64) error {
	o.granule += int64(opusSamples(data))
	return o.page(w, data, 0, o.granule)
}

func (o *oggOpus) end(io.Writer) error { return nil }

// page writes one packet as one Ogg page.
func (o *oggOpus) page(w io.Writer, packet []byte, kind byte, granule int64) error {
	var lacing []byte
	n := len(packet)
	for n >= 255 {
		lacing = append(lacing, 255)
		n -= 255
	}
	lacing = append(lacing, byte(n))
	if len(lacing) > 255 {
		return errors.New("an Opus packet too large for one page")
	}
	h := make([]byte, 27, 27+len(lacing))
	copy(h, "OggS")
	h[5] = kind
	binary.LittleEndian.PutUint64(h[6:], uint64(granule))
	binary.LittleEndian.PutUint32(h[14:], 0x53574159) // the stream's serial
	binary.LittleEndian.PutUint32(h[18:], o.seq)
	h[26] = byte(len(lacing))
	h = append(h, lacing...)
	o.seq++
	sum := oggUpdate(0, h)
	sum = oggUpdate(sum, packet)
	binary.LittleEndian.PutUint32(h[22:], sum)
	if _, err := w.Write(h); err != nil {
		return err
	}
	_, err := w.Write(packet)
	return err
}

// oggUpdate is Ogg's CRC: the 0x04C11DB7 polynomial, not reflected,
// which hash/crc32 does not make.
func oggUpdate(crc uint32, b []byte) uint32 {
	for _, c := range b {
		crc = crc<<8 ^ oggTable[byte(crc>>24)^c]
	}
	return crc
}

var oggTable = func() (t [256]uint32) {
	for i := range t {
		r := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if r&0x80000000 != 0 {
				r = r<<1 ^ 0x04C11DB7
			} else {
				r <<= 1
			}
		}
		t[i] = r
	}
	return
}()

// opusSamples is how many 48 kHz samples an Opus packet holds, read from
// its first byte (RFC 6716, 3.1).
func opusSamples(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	cfg := int(p[0] >> 3)
	var tenths int // of a millisecond
	switch {
	case cfg < 12:
		tenths = []int{100, 200, 400, 600}[cfg%4]
	case cfg < 16:
		tenths = []int{100, 200}[cfg%2]
	default:
		tenths = []int{25, 50, 100, 200}[cfg%4]
	}
	frames := 1
	switch p[0] & 3 {
	case 1, 2:
		frames = 2
	case 3:
		if len(p) > 1 {
			frames = int(p[1] & 0x3F)
		}
	}
	return frames * tenths * 48 / 10
}
