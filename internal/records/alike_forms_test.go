package records_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
)

// Records called the same are alike whatever form their letters are
// stored in: a task made on a Mac (é as e and an accent) and one made on
// Windows (é as one letter), and STRASSE and Straße.
func TestRecordsAlikeInAnyForm(t *testing.T) {
	t.Parallel()
	b := newBook(t)
	task := mustType(t, b, "task")
	one := made(t, b, "task", map[string]any{"title": "Caf\u00e9 run"})
	two := made(t, b, "task", map[string]any{"title": "Cafe\u0301 run"})
	made(t, b, "task", map[string]any{"title": "Cafe run"})
	if got := ids(records.Alike(b.Store, task, one)); len(got) != 1 || !got[two.ID] {
		t.Errorf("alike: only the other Café run, not Cafe run: %v", got)
	}
}
