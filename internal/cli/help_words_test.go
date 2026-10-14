package cli

import (
	"strings"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/fold"
)

// Each command is said in the usage text, so help and a typo's suggestion
// never name one the other does not know.
func TestEveryCommandIsInTheUsage(t *testing.T) {
	t.Parallel()
	for name := range commands {
		if name != "at-login" && !strings.Contains(usage, "  sameway "+name+" ") {
			t.Errorf("%s is not in the usage text", name)
		}
	}
}

func TestNearestIsCloseOrNothing(t *testing.T) {
	t.Parallel()
	for word, want := range map[string]string{"descrbe": "describe", "serach": "search", "ad": "", "zzz": "", "notes": "note"} {
		if got := fold.Nearest(word, []string{"describe", "search", "add", "note"}); got != want {
			t.Errorf("nearest(%q) = %q, want %q", word, got, want)
		}
	}
}
