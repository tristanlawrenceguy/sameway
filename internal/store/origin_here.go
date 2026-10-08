package store

import (
	"path/filepath"
	"strings"
)

// A copy of a workspace in another folder is another copy: a copy brought
// back on a new computer, or opened from the cloud folder to be kept in
// step with the first (internal/cloudsync). Each copy names itself in its
// stamps, so a database opened somewhere it was not before takes a name
// of its own; two copies under one name would each take the other's
// changes for its own and never pass them on.
func (s *Store) originHere(path string) {
	if path == ":memory:" {
		return
	}
	at, err := filepath.Abs(path)
	if err != nil {
		return
	}
	was := s.Meta("origin_at")
	if was != "" && !strings.EqualFold(was, at) {
		s.origin = NewID()
		s.db.Exec(`INSERT OR REPLACE INTO _meta (key, value) VALUES ('origin', ?)`, s.origin)
	}
	if was != at {
		s.SetMeta("origin_at", at)
	}
}

// OriginOf is the copy a stamp's clock names.
func OriginOf(clock string) string { return originOf(clock) }
