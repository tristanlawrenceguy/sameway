package speech

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Measured against a real meeting with its official annotations: one
// meeting of the AMI corpus (University of Edinburgh, CC BY 4.0), its room
// mix and each person's headset, and pyannote's RTTM of who spoke when.
// SAMEWAY_AMI_DIR holds <meeting>.Mix-Headset.wav, .Headset-0..3.wav and
// .rttm; SAMEWAY_AMI_MEETING names it (ES2004a when left out). Not in CI:
// it needs about 170 MB of audio and, for speakers, the models.

type span struct {
	start, end float64
	who        string
}

func amiFiles(t *testing.T) (dir, meeting string, ref []span) {
	dir = os.Getenv("SAMEWAY_AMI_DIR")
	if dir == "" {
		t.Skip("set SAMEWAY_AMI_DIR to a folder with an AMI meeting and its RTTM")
	}
	meeting = os.Getenv("SAMEWAY_AMI_MEETING")
	if meeting == "" {
		meeting = "ES2004a"
	}
	f, err := os.Open(filepath.Join(dir, meeting+".rttm"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		p := strings.Fields(sc.Text())
		if len(p) < 8 || p[0] != "SPEAKER" {
			continue
		}
		s, _ := strconv.ParseFloat(p[3], 64)
		d, _ := strconv.ParseFloat(p[4], 64)
		ref = append(ref, span{s, s + d, p[7]})
	}
	return dir, meeting, ref
}

func readSamples(t *testing.T, path string) []float32 {
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s, rate, err := ReadWAV(f)
	if err != nil || rate != Rate {
		t.Fatalf("%s: %v (rate %d)", path, err, rate)
	}
	return s
}

// der is the diarization error rate, frame by frame every 10 ms with
// overlapping speech counted: missed, false and confused speech over all
// the speech there was, with found voices matched to the people who make
// it least wrong. No collar, as AMI's published results score it.
func der(ref []span, hyp []Turn) float64 {
	const step = 0.01
	end := 0.0
	for _, r := range ref {
		end = math.Max(end, r.end)
	}
	n := int(end/step) + 1
	people := map[string]int{}
	for _, r := range ref {
		if _, ok := people[r.who]; !ok {
			people[r.who] = len(people)
		}
	}
	refAt := make([][]int, n)
	for _, r := range ref {
		for i := int(r.start / step); i < int(r.end/step) && i < n; i++ {
			refAt[i] = append(refAt[i], people[r.who])
		}
	}
	voices := 0
	hypAt := make([][]int, n)
	for _, h := range hyp {
		voices = max(voices, h.Speaker)
		for i := int(h.Start / step); i < int(h.End/step) && i < n; i++ {
			hypAt[i] = append(hypAt[i], h.Speaker-1)
		}
	}
	// Each frame's errors are its larger count less what is matched right,
	// and what is matched right is how long each person's matched voice
	// overlaps them: so the best matching is found from those overlaps.
	worst, total := 0, 0
	overlap := make([][]int, len(people))
	for p := range overlap {
		overlap[p] = make([]int, voices)
	}
	for i := 0; i < n; i++ {
		total += len(refAt[i])
		worst += max(len(refAt[i]), len(hypAt[i]))
		for _, r := range refAt[i] {
			for _, h := range hypAt[i] {
				overlap[r][h]++
			}
		}
	}
	right := 0
	used := make([]bool, voices)
	var try func(p, sum int)
	try = func(p, sum int) {
		if p == len(people) {
			right = max(right, sum)
			return
		}
		try(p+1, sum) // nobody's voice for this person
		for v := 0; v < voices; v++ {
			if !used[v] {
				used[v] = true
				try(p+1, sum+overlap[p][v])
				used[v] = false
			}
		}
	}
	try(0, 0)
	return float64(worst-right) / float64(total)
}

// Speakers are told apart in a real four-person meeting, and how well is
// said as its diarization error rate.
func TestSpeakersOnAMI(t *testing.T) {
	t.Parallel()
	dir, meeting, ref := amiFiles(t)
	kit := os.Getenv("SAMEWAY_SPEAKERS_DIR")
	if kit == "" {
		t.Skip("set SAMEWAY_SPEAKERS_DIR to the engine and speaker models too")
	}
	for _, n := range []int{0, 4} {
		turns, err := Diarize(context.Background(), kit, filepath.Join(dir, meeting+".Mix-Headset.wav"), n)
		if err != nil {
			t.Fatal(err)
		}
		voices := map[int]bool{}
		for _, tr := range turns {
			voices[tr.Speaker] = true
		}
		told := "not told how many"
		if n > 0 {
			told = fmt.Sprintf("told %d", n)
		}
		rate := der(ref, turns)
		t.Logf("%s, %s: %d voices found, diarization error rate %.1f%%", meeting, told, len(voices), 100*rate)
		// Published systems on the same segmentation score about 20%;
		// the voice model was chosen for meeting that, and must keep to it.
		if n > 0 && rate > 0.30 {
			t.Errorf("told how many spoke, the error rate should stay under 30%%: %.1f%%", 100*rate)
		}
	}
}

// meThem is the page's rule (design/base/28-speech.js), second by second:
// t when the call sounds, m when only the microphone does.
func meThem(me, them []float32) string {
	const window = Rate / 4 // 250 ms
	rms := func(s []float32, from int) float64 {
		sum := 0.0
		for _, v := range s[from : from+window] {
			sum += float64(v) * float64(v)
		}
		return math.Sqrt(sum / window)
	}
	var b strings.Builder
	m, th, k := 0, 0, 0
	for at := 0; at+window <= len(me) && at+window <= len(them); at += window {
		if rms(them, at) > 0.002 {
			th++
		} else if rms(me, at) > 0.003 {
			m++
		}
		if k++; k == 4 {
			switch {
			case th > 0 && th >= m:
				b.WriteByte('t')
			case m > 0:
				b.WriteByte('m')
			default:
				b.WriteByte('.')
			}
			m, th, k = 0, 0, 0
		}
	}
	return b.String()
}

// Me and them, measured: one person's headset is the microphone and the
// other three the call; each second where the annotations say only that
// person, or only the others, spoke is checked, with the microphone clean
// (headphones) and hearing the call (speakers).
func TestMeAndThemOnAMI(t *testing.T) {
	t.Parallel()
	dir, meeting, ref := amiFiles(t)
	var heads [4][]float32
	for i := range heads {
		heads[i] = readSamples(t, filepath.Join(dir, fmt.Sprintf("%s.Headset-%d.wav", meeting, i)))
	}
	n := len(heads[0])
	// Which person each headset is: the one it is loudest for.
	who := whoseHeadset(heads, ref)
	t.Logf("headset 0 is %s", who)
	them := make([]float32, n)
	for i := 1; i < 4; i++ {
		for j := 0; j < n && j < len(heads[i]); j++ {
			them[j] += heads[i][j]
		}
	}
	for _, room := range []float32{0, 0.3} {
		mic := make([]float32, n)
		for j := range mic {
			mic[j] = heads[0][j] + room*them[j]
		}
		said := meThem(mic, them)
		right, checked := 0, 0
		for sec := 0; sec < len(said); sec++ {
			mine, others := false, false
			for _, r := range ref {
				if r.start < float64(sec+1) && r.end > float64(sec) {
					if r.who == who {
						mine = true
					} else {
						others = true
					}
				}
			}
			if mine == others {
				continue // nobody, or both at once
			}
			checked++
			if mine && said[sec] == 'm' || others && said[sec] == 't' {
				right++
			}
		}
		t.Logf("%s, call heard by the microphone at %.0f%%: %d of %d seconds right (%.1f%%)", meeting, room*100, right, checked, 100*float64(right)/float64(checked))
		if float64(right) < 0.9*float64(checked) {
			t.Errorf("me and them should be right most of the time: %d of %d", right, checked)
		}
	}
}

func whoseHeadset(heads [4][]float32, ref []span) string {
	loud := map[string]float64{}
	for _, r := range ref {
		from, to := int(r.start*Rate), int(r.end*Rate)
		if to > len(heads[0]) {
			to = len(heads[0])
		}
		for j := from; j < to; j++ {
			loud[r.who] += float64(heads[0][j]) * float64(heads[0][j])
		}
	}
	dur := map[string]float64{}
	for _, r := range ref {
		dur[r.who] += r.end - r.start
	}
	best, most := "", -1.0
	for w, e := range loud {
		if e/dur[w] > most {
			best, most = w, e/dur[w]
		}
	}
	return best
}
