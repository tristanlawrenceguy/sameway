package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A workspace is copied once a day, the last seven are kept, and a copy
// put back keeps the data from before, so the restore can be undone too.
func TestSnapshotsAreDailyKeptForAWeekAndRestoreKeepsWhatWasThere(t *testing.T) {
	t.Setenv("SAMEWAY_KNOWN", filepath.Join(t.TempDir(), "known.json"))
	w := &Workspace{Dir: t.TempDir()}
	os.WriteFile(w.DBPath(), []byte("today"), 0o644)
	copyDB := func(path string) error { return copyFile(w.DBPath(), path) }

	day := time.Date(2026, 9, 1, 9, 0, 0, 0, time.Local)
	for i := 0; i < 10; i++ {
		if _, err := w.DailySnapshot(day.AddDate(0, 0, i), copyDB); err != nil {
			t.Fatal(err)
		}
	}
	if again, _ := w.DailySnapshot(day.AddDate(0, 0, 9), copyDB); again != "" {
		t.Error("one copy a day: a second on the same day is not made")
	}
	if n := len(w.Snapshots()); n != KeepSnapshots {
		t.Fatalf("the last %d are kept, got %d", KeepSnapshots, n)
	}

	old := filepath.Join(w.SnapshotDir(), "data-2026-09-10.db")
	os.WriteFile(old, []byte("yesterday"), 0o644)
	kept, err := w.Restore(old)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(w.DBPath()); string(got) != "yesterday" {
		t.Errorf("the copy is back, got %q", got)
	}
	if before, _ := os.ReadFile(kept); string(before) != "today" {
		t.Errorf("what was there is kept, got %q at %s", before, kept)
	}
}
