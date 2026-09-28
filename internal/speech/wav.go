package speech

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// The engine reads 16 kHz mono 16-bit WAV. A page's script turns any
// recording its browser can play into that; this does the same for a WAV
// that arrives any other way, at any rate and with any number of channels,
// so a recording can be written down with no script and no other program.

// Rate is the sample rate the engine reads.
const Rate = 16000

// ReadWAV reads a PCM (8, 16, 24 or 32-bit) or float WAV into mono samples
// between -1 and 1, and its sample rate.
func ReadWAV(r io.Reader) ([]float32, int, error) {
	f, err := wavFormat(r)
	if err != nil {
		return nil, 0, err
	}
	data, err := io.ReadAll(io.LimitReader(r, f.size))
	if err != nil {
		return nil, 0, err
	}
	samples, err := mono(data, f.format, f.channels, f.bits)
	return samples, f.rate, err
}

// wavInfo is what a WAV's header says of its sound.
type wavInfo struct {
	format         uint16
	channels, bits int
	rate           int
	size           int64 // bytes of sound
}

// wavFormat reads a WAV's header and leaves r at the start of its sound.
func wavFormat(r io.Reader) (wavInfo, error) {
	var head [12]byte
	if _, err := io.ReadFull(r, head[:]); err != nil || string(head[0:4]) != "RIFF" || string(head[8:12]) != "WAVE" {
		return wavInfo{}, errors.New("this is not a WAV file")
	}
	var f wavInfo
	for {
		var ch [8]byte
		if _, err := io.ReadFull(r, ch[:]); err != nil {
			return wavInfo{}, errors.New("the WAV file has no sound in it")
		}
		size := binary.LittleEndian.Uint32(ch[4:8])
		switch string(ch[0:4]) {
		case "fmt ":
			buf := make([]byte, size)
			if _, err := io.ReadFull(r, buf); err != nil || size < 16 {
				return wavInfo{}, errors.New("the WAV file's format is cut short")
			}
			f.format, f.channels = binary.LittleEndian.Uint16(buf[0:2]), int(binary.LittleEndian.Uint16(buf[2:4]))
			f.rate, f.bits = int(binary.LittleEndian.Uint32(buf[4:8])), int(binary.LittleEndian.Uint16(buf[14:16]))
			if f.format == 0xFFFE && size >= 26 {
				f.format = binary.LittleEndian.Uint16(buf[24:26])
			}
			if size%2 == 1 {
				io.CopyN(io.Discard, r, 1)
			}
		case "data":
			if f.channels == 0 || f.rate == 0 {
				return wavInfo{}, errors.New("the WAV file's sound comes before its format")
			}
			f.size = int64(size)
			// A recording still being written says its size is all of it.
			if size == 0 || size == 0xFFFFFFFF {
				f.size = 1 << 62
			}
			return f, nil
		default:
			if _, err := io.CopyN(io.Discard, r, int64(size)+int64(size%2)); err != nil {
				return wavInfo{}, errors.New("the WAV file is cut short")
			}
		}
	}
}

// mono averages a frame's channels into one sample.
func mono(data []byte, format uint16, channels, bits int) ([]float32, error) {
	width := bits / 8
	if width == 0 || (format != 1 && format != 3) || (format == 3 && bits != 32) {
		return nil, fmt.Errorf("this WAV's kind of sound (format %d, %d bits) cannot be read here", format, bits)
	}
	frame := width * channels
	out := make([]float32, 0, len(data)/frame)
	for i := 0; i+frame <= len(data); i += frame {
		var sum float64
		for c := 0; c < channels; c++ {
			b := data[i+c*width : i+(c+1)*width]
			switch {
			case format == 3:
				sum += float64(math.Float32frombits(binary.LittleEndian.Uint32(b)))
			case width == 1:
				sum += (float64(b[0]) - 128) / 128
			case width == 2:
				sum += float64(int16(binary.LittleEndian.Uint16(b))) / 32768
			case width == 3:
				sum += float64(int32(uint32(b[0])<<8|uint32(b[1])<<16|uint32(b[2])<<24)>>8) / 8388608
			case width == 4:
				sum += float64(int32(binary.LittleEndian.Uint32(b))) / 2147483648
			}
		}
		out = append(out, float32(sum/float64(channels)))
	}
	return out, nil
}

// Resample brings samples to the engine's rate, by straight lines between
// them: plenty for speech.
func Resample(in []float32, rate int) []float32 {
	if rate == Rate || len(in) == 0 {
		return in
	}
	n := int(float64(len(in)) * Rate / float64(rate))
	out := make([]float32, n)
	step := float64(rate) / Rate
	for i := range out {
		pos := float64(i) * step
		j := int(pos)
		if j+1 >= len(in) {
			out[i] = in[len(in)-1]
			continue
		}
		f := float32(pos - float64(j))
		out[i] = in[j]*(1-f) + in[j+1]*f
	}
	return out
}

// WriteWAV writes 16 kHz mono 16-bit samples as a WAV.
func WriteWAV(w io.Writer, samples []float32) error {
	data := len(samples) * 2
	head := make([]byte, 44)
	copy(head[0:], "RIFF")
	binary.LittleEndian.PutUint32(head[4:], uint32(36+data))
	copy(head[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(head[16:], 16)
	binary.LittleEndian.PutUint16(head[20:], 1)
	binary.LittleEndian.PutUint16(head[22:], 1)
	binary.LittleEndian.PutUint32(head[24:], Rate)
	binary.LittleEndian.PutUint32(head[28:], Rate*2)
	binary.LittleEndian.PutUint16(head[32:], 2)
	binary.LittleEndian.PutUint16(head[34:], 16)
	copy(head[36:], "data")
	binary.LittleEndian.PutUint32(head[40:], uint32(data))
	if _, err := w.Write(head); err != nil {
		return err
	}
	buf := make([]byte, 2*4096)
	for i := 0; i < len(samples); i += 4096 {
		end := i + 4096
		if end > len(samples) {
			end = len(samples)
		}
		n := 0
		for _, s := range samples[i:end] {
			v := math.Max(-1, math.Min(1, float64(s)))
			binary.LittleEndian.PutUint16(buf[n:], uint16(int16(v*32767)))
			n += 2
		}
		if _, err := w.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}
