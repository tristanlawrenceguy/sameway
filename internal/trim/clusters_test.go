package trim

import (
	"strings"
	"testing"
)

// A title is never cut inside what a person sees as one character: a
// family emoji stays whole or goes, a flag keeps both its letters, a skin
// tone stays with its hand, an accent written as its own mark stays with
// its letter. Swept across every place the cut can fall.
func TestACutKeepsEveryCharacterWhole(t *testing.T) {
	t.Parallel()
	family := "👨‍👩‍👧"
	flag := "🇵🇹"
	wave := "👋🏽"
	accent := "é" // é as e and a combining acute
	whole := []string{family, flag, wave, accent}
	for _, piece := range whole {
		for pad := 40; pad <= 80; pad++ {
			s := strings.Repeat("a", pad) + strings.Repeat(piece, 4)
			for name, got := range map[string]string{"Title": Title(s), "Clip": Clip(s, 50), "Line": Line(s, 60)} {
				kept := strings.TrimSuffix(got, "…")
				rest := strings.TrimPrefix(kept, strings.Repeat("a", min(pad, len(kept))))
				if strings.Trim(rest, piece) != "" && !strings.HasPrefix(s, kept) {
					t.Fatalf("%s of %d a's and %q cut it: %q", name, pad, piece, got)
				}
				if trimmed := strings.TrimLeft(rest, "a"); strings.ReplaceAll(trimmed, piece, "") != "" {
					t.Errorf("%s of %d a's and %q cut inside it: %q", name, pad, piece, got)
				}
			}
		}
	}
	if got := Clip("plain words that run on and on", 10); got != "plain wor…" {
		t.Errorf("plain text is cut as before: %q", got)
	}
}
