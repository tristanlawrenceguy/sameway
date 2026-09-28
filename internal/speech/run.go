package speech

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
)

// runner is the engine's program that listens for speech and writes down
// each stretch of it.
func runner(dir string) string {
	name := "sherpa-onnx-vad-with-offline-asr"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, "engine", "bin", name)
}

// Transcribe writes down what is said in a 16 kHz mono WAV, a stretch at a
// time, with when each was said. The language is Whisper's to tell.
func Transcribe(ctx context.Context, dir, wav string) ([]convert.Cue, error) {
	if !Ready(dir) {
		return nil, fmt.Errorf("speech-to-text is not on this computer yet")
	}
	threads := runtime.NumCPU() / 2
	if threads < 1 {
		threads = 1
	}
	if threads > 4 {
		threads = 4
	}
	cmd := exec.CommandContext(ctx, runner(dir),
		"--silero-vad-model="+filepath.Join(dir, "vad.onnx"),
		"--whisper-encoder="+filepath.Join(dir, "encoder.onnx"),
		"--whisper-decoder="+filepath.Join(dir, "decoder.onnx"),
		"--tokens="+filepath.Join(dir, "tokens.txt"),
		"--num-threads="+strconv.Itoa(threads),
		wav)
	lib := filepath.Join(dir, "engine", "lib")
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+lib, "DYLD_LIBRARY_PATH="+lib)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("the speech engine stopped: %v: %s", err, lastLine(out.String()))
	}
	return Parse(out.String()), nil
}

var segment = regexp.MustCompile(`^(\d+(?:\.\d+)?) -- (\d+(?:\.\d+)?): ?(.*)$`)

// Parse reads the engine's lines of what was said: "0.326 -- 16.704: words".
func Parse(out string) []convert.Cue {
	var cues []convert.Cue
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		m := segment.FindStringSubmatch(strings.TrimSpace(sc.Text()))
		if m == nil {
			continue
		}
		text := strings.TrimSpace(m[3])
		if text == "" {
			continue
		}
		start, _ := strconv.ParseFloat(m[1], 64)
		end, _ := strconv.ParseFloat(m[2], 64)
		cues = append(cues, convert.Cue{Start: start, End: end, Text: text})
	}
	return cues
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// VTT writes cues as a WebVTT file.
func VTT(cues []convert.Cue) string {
	var b strings.Builder
	b.WriteString("WEBVTT\n")
	for _, c := range cues {
		fmt.Fprintf(&b, "\n%s --> %s\n", stamp(c.Start), stamp(c.End))
		if c.Speaker != "" {
			b.WriteString("<v " + c.Speaker + ">")
		}
		b.WriteString(c.Text + "\n")
	}
	return b.String()
}

func stamp(s float64) string {
	ms := int(s*1000 + 0.5)
	return fmt.Sprintf("%02d:%02d:%02d.%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}
