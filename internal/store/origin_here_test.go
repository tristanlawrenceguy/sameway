package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A database opened again where it was keeps its name; a copy of it
// opened in another folder takes a name of its own.
func TestACopyElsewhereNamesItselfApart(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	first := filepath.Join(dir, "a", "data.db")
	s, err := openAt(t, first)
	if err != nil {
		t.Fatal(err)
	}
	origin := s.Origin()
	copyAt := filepath.Join(dir, "b.db")
	if err := s.Backup(copyAt); err != nil {
		t.Fatal(err)
	}
	s.Close()
	again, _ := openAt(t, first)
	if again.Origin() != origin {
		t.Error("opened again in its place, it keeps its name")
	}
	again.Close()
	c, _ := openAt(t, copyAt)
	defer c.Close()
	if c.Origin() == origin || c.Origin() == "" {
		t.Errorf("the copy names itself apart: %q", c.Origin())
	}
}

func openAt(t *testing.T, path string) (*Store, error) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return Open(path, &schema.Set{})
}
