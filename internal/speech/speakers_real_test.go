package speech

import (
	"context"
	"os"
	"testing"
)

// With SAMEWAY_SPEAKERS_DIR set to a folder with the engine and the two
// speaker models, and SAMEWAY_SPEAKERS_WAV to a 16 kHz recording of two
// voices, the real program tells them apart.
func TestSpeakersAreToldApartForReal(t *testing.T) {
	t.Parallel()
	dir, wav := os.Getenv("SAMEWAY_SPEAKERS_DIR"), os.Getenv("SAMEWAY_SPEAKERS_WAV")
	if dir == "" || wav == "" {
		t.Skip("set SAMEWAY_SPEAKERS_DIR and SAMEWAY_SPEAKERS_WAV to run the real program")
	}
	turns, err := Diarize(context.Background(), dir, wav, 2)
	if err != nil {
		t.Fatal(err)
	}
	voices := map[int]bool{}
	for _, tr := range turns {
		voices[tr.Speaker] = true
	}
	t.Logf("%d turns: %+v", len(turns), turns)
	if len(voices) != 2 {
		t.Errorf("two voices, told apart: %+v", turns)
	}
}
