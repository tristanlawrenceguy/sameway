package speech

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// A long WAV is cut into 16 kHz mono chunks on this computer, a second at
// a time, each a WAV the engine reads, with the sound carried across.
func TestAWAVIsCutIntoChunksAtHome(t *testing.T) {
	t.Parallel()
	rate, secs := 44100, 3
	var data bytes.Buffer
	for i := 0; i < rate*secs; i++ {
		v := int16(12000 * math.Sin(2*math.Pi*440*float64(i)/float64(rate)))
		binary.Write(&data, binary.LittleEndian, v) // left
		binary.Write(&data, binary.LittleEndian, v) // right
	}
	var file bytes.Buffer
	file.WriteString("RIFF")
	binary.Write(&file, binary.LittleEndian, uint32(36+data.Len()))
	file.WriteString("WAVEfmt ")
	binary.Write(&file, binary.LittleEndian, []uint32{16})
	binary.Write(&file, binary.LittleEndian, []uint16{1, 2})
	binary.Write(&file, binary.LittleEndian, []uint32{uint32(rate), uint32(rate * 4)})
	binary.Write(&file, binary.LittleEndian, []uint16{4, 16})
	file.WriteString("data")
	binary.Write(&file, binary.LittleEndian, uint32(data.Len()))
	file.Write(data.Bytes())
	dir := t.TempDir()
	src := filepath.Join(dir, "long.wav")
	os.WriteFile(src, file.Bytes(), 0o644)

	parts, err := SplitWAV(src, filepath.Join(dir, "parts"), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 3 || parts[1].Start != 1 || parts[2].Start != 2 {
		t.Fatalf("three chunks of a second: %+v", parts)
	}
	total := 0
	for _, p := range parts {
		b, _ := os.ReadFile(p.Path)
		samples, r, err := ReadWAV(bytes.NewReader(b))
		if err != nil || r != Rate {
			t.Fatalf("each chunk is a 16 kHz WAV: %v %d", err, r)
		}
		total += len(samples)
		peak := float32(0)
		for _, s := range samples {
			peak = max(peak, s)
		}
		if peak < 0.3 || peak > 0.4 {
			t.Errorf("the sound is carried across, peak %v", peak)
		}
	}
	if total < 3*Rate-2 || total > 3*Rate {
		t.Errorf("three seconds at 16 kHz is about 48000 samples, got %d", total)
	}
}
