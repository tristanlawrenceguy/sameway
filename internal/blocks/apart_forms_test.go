package blocks_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
)

// Two titles that look the same are told apart though their letters are
// stored differently: Café typed as one letter, and as e and an accent.
// A screen reader says both alike, so they are one name twice.
func TestTitlesThatLookTheSameAreToldApart(t *testing.T) {
	t.Parallel()
	names := []string{"Caf\u00e9", "Cafe\u0301", "Cafe"}
	said := blocks.Apart(names, func(i int) []string { return []string{[]string{"due today", "due Friday", "x"}[i]} })
	if said[0] != "due today" || said[1] != "due Friday" || said[2] != "" {
		t.Errorf("told apart: %q", said)
	}
}
