package speech

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Telling speakers apart: the same engine's diarisation program, with a
// model that finds where one voice gives way to another (pyannote
// segmentation 3.0) and one that tells voices apart (3D-Speaker
// ERes2Net, trained on English speech). Chosen by measuring: on an AMI
// meeting with its official annotations (ami_bench_test.go) it put 20% of
// speech wrong, told how many spoke, where the other voice models the
// engine's makers publish put 31% to 114% wrong. Like speech-to-text, it is fetched once,
// when the owner asks, from where its makers publish it, each checked
// against the fingerprint written here; and the recording never leaves.

const models = "https://github.com/k2-fsa/sherpa-onnx/releases/download/"

var speakerModels = []asset{
	{models + "speaker-segmentation-models/sherpa-onnx-pyannote-segmentation-3-0.tar.bz2", "24615ee884c897d9d2ba09bb4d30da6bb1b15e685065962db5b02e76e4996488", 6958444, "speakers-segmentation.tar.bz2"},
	{models + "speaker-recongition-models/3dspeaker_speech_eres2net_sv_en_voxceleb_16k.onnx", "c59158379255ad66e161679cca6af8d52d51e389e3224ab7d7a7baae295c2db5", 26485263, "speakers-voices.onnx"},
}

// SpeakersSize is how much telling speakers apart downloads.
func SpeakersSize() int64 { return speakerModels[0].Size + speakerModels[1].Size }

func segmentation(dir string) string {
	return filepath.Join(dir, "speakers-segmentation", "model.onnx")
}

func diariser(dir string) string {
	name := "sherpa-onnx-offline-speaker-diarization"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(dir, "engine", "bin", name)
}

// SpeakersReady says whether speakers can be told apart here.
func SpeakersReady(dir string) bool {
	if _, err := os.Stat(diariser(dir)); err != nil {
		return false
	}
	if _, err := os.Stat(segmentation(dir)); err != nil {
		return false
	}
	st, err := os.Stat(filepath.Join(dir, speakerModels[1].Name))
	return err == nil && st.Size() == speakerModels[1].Size
}

// InstallSpeakers fetches the two models, once, into dir, beside the
// engine speech-to-text already brought.
func InstallSpeakers(ctx context.Context, client *http.Client, dir string, progress Progress) error {
	if _, err := os.Stat(diariser(dir)); err != nil {
		return fmt.Errorf("speech-to-text is not on this computer yet, and telling speakers apart needs it first")
	}
	if client == nil {
		client = http.DefaultClient
	}
	total, done := SpeakersSize(), int64(0)
	for _, a := range speakerModels {
		path := filepath.Join(dir, a.Name)
		if !sameFile(path, a.SHA256) && !(a == speakerModels[0] && fileExists(segmentation(dir))) {
			base := done
			if err := fetch(ctx, client, a, path, func(n int64) {
				if progress != nil {
					progress(base+n, total)
				}
			}); err != nil {
				return err
			}
		}
		done += a.Size
	}
	if !fileExists(segmentation(dir)) {
		out := filepath.Join(dir, "speakers-segmentation")
		os.RemoveAll(out)
		if err := unpack(filepath.Join(dir, speakerModels[0].Name), out); err != nil {
			os.RemoveAll(out)
			return fmt.Errorf("could not unpack the speaker model: %w", err)
		}
		os.Remove(filepath.Join(dir, speakerModels[0].Name))
	}
	if !SpeakersReady(dir) {
		return fmt.Errorf("the speaker models were fetched but are not whole")
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Turn is one stretch of a recording and which voice spoke it, numbered
// from 1 in the order the voices are first heard.
type Turn struct {
	Start, End float64
	Speaker    int
}

// Diarize says who spoke when in a 16 kHz mono WAV. speakers is how many
// there are when that is known, 0 when it is not.
func Diarize(ctx context.Context, dir, wav string, speakers int) ([]Turn, error) {
	if !SpeakersReady(dir) {
		return nil, fmt.Errorf("telling speakers apart is not on this computer yet")
	}
	// Not told how many: on the AMI meeting 1.1 found a few voices too
	// many (21.5% wrong) where 1.2 found four (20.1%) but 1.3 found only
	// one. Too many can be put together when they are named; one cannot
	// be pulled apart, so the setting stays clear of that edge.
	cluster := "--clustering.cluster-threshold=1.1"
	if speakers >= 2 {
		cluster = "--clustering.num-clusters=" + strconv.Itoa(speakers)
	}
	return diarizeWith(ctx, dir, wav, cluster, filepath.Join(dir, speakerModels[1].Name))
}

// diarizeWith runs the program with the clustering and voice model asked for.
func diarizeWith(ctx context.Context, dir, wav, cluster, voices string) ([]Turn, error) {
	cmd := exec.CommandContext(ctx, diariser(dir), cluster,
		"--segmentation.pyannote-model="+segmentation(dir),
		"--embedding.model="+voices,
		wav)
	lib := filepath.Join(dir, "engine", "lib")
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+lib, "DYLD_LIBRARY_PATH="+lib)
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("telling the speakers apart stopped: %v: %s", err, lastLine(out.String()))
	}
	return ParseTurns(out.String()), nil
}

var turnLine = regexp.MustCompile(`^(\d+(?:\.\d+)?) -- (\d+(?:\.\d+)?) speaker_(\d+)$`)

// ParseTurns reads the program's lines, "0.031 -- 2.039 speaker_00", and
// numbers the voices in the order they are first heard.
func ParseTurns(out string) []Turn {
	var turns []Turn
	order := map[string]int{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		m := turnLine.FindStringSubmatch(strings.TrimSpace(sc.Text()))
		if m == nil {
			continue
		}
		if _, ok := order[m[3]]; !ok {
			order[m[3]] = len(order) + 1
		}
		start, _ := strconv.ParseFloat(m[1], 64)
		end, _ := strconv.ParseFloat(m[2], 64)
		turns = append(turns, Turn{Start: start, End: end, Speaker: order[m[3]]})
	}
	return turns
}
