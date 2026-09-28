package speech

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// A WAV is already plain sound, so the computer that hosts the workspace
// writes it down itself, with no browser: read a second at a time, made
// one channel at 16 kHz, and cut into chunks the engine takes one by one.
// However long it is, only a second of it is ever held.

// Part is one chunk of plain sound: where it starts, in seconds, and its file.
type Part struct {
	Start float64
	Path  string
}

// SplitWAV writes the WAV at src into dir as 16 kHz mono chunks of every
// seconds each.
func SplitWAV(src, dir string, every float64) ([]Part, error) {
	f, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := bufio.NewReaderSize(f, 1<<16)
	info, err := wavFormat(r)
	if err != nil {
		return nil, err
	}
	frame := info.channels * info.bits / 8
	if frame == 0 {
		return nil, fmt.Errorf("this WAV's kind of sound (%d bits) cannot be read here", info.bits)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var parts []Part
	var out *os.File
	var written int64
	perPart := int64(every * Rate)
	finish := func() error {
		if out == nil {
			return nil
		}
		head := wavHead(written)
		_, err := out.WriteAt(head, 0)
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		out = nil
		return err
	}
	emit := func(samples []float32) error {
		for len(samples) > 0 {
			if out == nil || written >= perPart {
				if err := finish(); err != nil {
					return err
				}
				path := filepath.Join(dir, fmt.Sprintf("%d.wav", len(parts)))
				o, err := os.Create(path)
				if err != nil {
					return err
				}
				out, written = o, 0
				o.Write(wavHead(0))
				parts = append(parts, Part{Start: float64(len(parts)) * every, Path: path})
			}
			n := min(int64(len(samples)), perPart-written)
			buf := make([]byte, 2*n)
			for i, s := range samples[:n] {
				v := max(-1, min(1, float64(s)))
				binary.LittleEndian.PutUint16(buf[2*i:], uint16(int16(v*32767)))
			}
			if _, err := out.Write(buf); err != nil {
				return err
			}
			written += n
			samples = samples[n:]
		}
		return nil
	}
	// Straight lines between samples, across the seams between seconds.
	step := float64(info.rate) / Rate
	next, base := 0.0, int64(0) // the next output's place, and the block's first input
	var prev []float32
	block := make([]byte, frame*info.rate)
	left := info.size
	for left > 0 {
		n, err := io.ReadFull(r, block[:min(int64(len(block)), left)/int64(frame)*int64(frame)])
		if n == 0 {
			break
		}
		left -= int64(n)
		in, merr := mono(block[:n], info.format, info.channels, info.bits)
		if merr != nil {
			return nil, merr
		}
		// The last sample of the block before goes first, so a line can
		// run from it into this one.
		all := append(prev, in...)
		from := base - int64(len(prev))
		var outs []float32
		for {
			i := int64(next) - from
			if i+1 >= int64(len(all)) {
				break
			}
			f := float32(next - float64(int64(next)))
			outs = append(outs, all[i]*(1-f)+all[i+1]*f)
			next += step
		}
		if err := emit(outs); err != nil {
			return nil, err
		}
		prev = append(prev[:0], all[len(all)-1])
		base += int64(len(in))
		if err != nil {
			break
		}
	}
	if err := finish(); err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("the WAV file has no sound in it")
	}
	return parts, nil
}

// wavHead is the header of a 16 kHz mono 16-bit WAV of n samples.
func wavHead(n int64) []byte {
	var b bytes44
	WriteWAV(&b, nil)
	h := b.b
	binary.LittleEndian.PutUint32(h[4:], uint32(36+2*n))
	binary.LittleEndian.PutUint32(h[40:], uint32(2*n))
	return h
}

type bytes44 struct{ b []byte }

func (w *bytes44) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}
