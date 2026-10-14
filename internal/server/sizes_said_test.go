package server_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
)

// A size is said in MB from 1 MB, with a tenth below 10 MB so 1.9 MB is
// not 2, in KB below, and never as a count of bytes; a download says its
// size only from 1 MB, where a phone or a mail would notice.
func TestSizesAreSaidAsAPersonDecidesOnThem(t *testing.T) {
	t.Parallel()
	for n, want := range map[int64][2]string{
		59:                 {"under 1 KB", ""},
		4096:               {"4 KB", ""},
		1 << 20:            {"1 MB", "1 MB"},
		1<<20 + 9<<17:      {"2.1 MB", "2.1 MB"},
		38 << 20:           {"38 MB", "38 MB"},
		1<<20 - 1:          {"1 MB", "1 MB"},
		10<<20 - (1 << 10): {"10 MB", "10 MB"},
	} {
		if got := blocks.SizeWords(n); got != want[0] {
			t.Errorf("SizeWords(%d) = %q, want %q", n, got, want[0])
		}
		if got := blocks.DownloadSize(n); got != want[1] {
			t.Errorf("DownloadSize(%d) = %q, want %q", n, got, want[1])
		}
	}
}
