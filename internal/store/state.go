package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// A workspace can be hosted by more than one computer at once, each with
// its own database, kept the same by sync. What makes that work is kept
// here, beside the tables, in _state: every field of every shared record,
// with the moment it was last written, stamped by a hybrid logical clock
// that also names the computer that wrote it. Two computers that have
// seen the same stamps hold the same records, whatever order the stamps
// came in: for each field, the latest stamp wins. So two people changing
// different fields of one note both keep their change, and a record is
// deleted by a field too (_deleted), which a later undo can win over.
//
// The ordinary tables stay what they were, the readable form: they are
// written from _state whenever a stamp from elsewhere wins.

// Stamp is one field of one record as some computer last wrote it.
type Stamp struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Field string          `json:"field"`
	Value json.RawMessage `json:"value"`
	Clock string          `json:"clock"`
}

const (
	fieldDeleted = "_deleted"
	fieldCreated = "_created"
)

func (s *Store) migrateState() error {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS _state (type TEXT NOT NULL, id TEXT NOT NULL, field TEXT NOT NULL, value TEXT, clock TEXT NOT NULL, PRIMARY KEY (type, id, field))`,
		`CREATE INDEX IF NOT EXISTS _state_clock ON _state (clock)`,
		`CREATE TABLE IF NOT EXISTS _seen (origin TEXT PRIMARY KEY, clock TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS _meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
	} {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("sync state: %w", err)
		}
	}
	if err := s.db.QueryRow(`SELECT value FROM _meta WHERE key = 'origin'`).Scan(&s.origin); err != nil {
		s.origin = NewID()
		if _, err := s.db.Exec(`INSERT INTO _meta (key, value) VALUES ('origin', ?)`, s.origin); err != nil {
			return err
		}
	}
	return nil
}

// Origin names this computer's copy of the workspace in every stamp.
func (s *Store) Origin() string { return s.origin }

// hlc is a hybrid logical clock: the wall clock in milliseconds, never
// going back, with a counter for writes within one millisecond. It moves
// past any stamp it hears of, so a stamp made after hearing of another
// always sorts after it, whatever the two computers' clocks say.
type hlc struct {
	mu      sync.Mutex
	wall    int64
	counter int
}

func (c *hlc) next(origin string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now().UnixMilli()
	if now > c.wall {
		c.wall, c.counter = now, 0
	} else {
		c.counter++
	}
	return fmt.Sprintf("%013x.%04x.%s", c.wall, c.counter, origin)
}

func (c *hlc) hear(clock string) {
	var wall int64
	var counter int
	if _, err := fmt.Sscanf(clock, "%x.%x.", &wall, &counter); err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if wall > c.wall || wall == c.wall && counter > c.counter {
		c.wall, c.counter = wall, counter
	}
}

// originOf is the computer a stamp came from.
func originOf(clock string) string {
	if i := strings.LastIndex(clock, "."); i >= 0 {
		return clock[i+1:]
	}
	return ""
}

// shared says whether a type's records are kept the same on every host.
func (s *Store) shared(typeName string) bool { return !s.Local[typeName] }

// local says whether one record stays on this computer although its type
// is shared, by LocalRecord.
func (s *Store) local(typeName string, fields map[string]any) bool {
	return s.LocalRecord != nil && fields != nil && s.LocalRecord(typeName, fields)
}

// stamp records a local write: the fields that changed from before to
// after (all of them for a new record, only _deleted for a removal).
func (s *Store) stamp(t *schema.Type, id string, before, after map[string]any, created time.Time) {
	if !s.shared(t.Name) || s.local(t.Name, after) || s.local(t.Name, before) {
		return
	}
	put := func(field string, v any) {
		raw, err := json.Marshal(v)
		if err != nil {
			return
		}
		clock := s.clock.next(s.origin)
		s.db.Exec(`INSERT OR REPLACE INTO _state (type, id, field, value, clock) VALUES (?, ?, ?, ?, ?)`, t.Name, id, field, string(raw), clock)
		s.db.Exec(`INSERT OR REPLACE INTO _seen (origin, clock) VALUES (?, ?)`, s.origin, clock)
	}
	if after == nil {
		put(fieldDeleted, true)
		return
	}
	if before == nil {
		put(fieldCreated, created.UTC().Format(time.RFC3339Nano))
		put(fieldDeleted, false)
	}
	for _, f := range t.Fields {
		a, _ := json.Marshal(after[f.Name])
		if before != nil {
			if b, _ := json.Marshal(before[f.Name]); string(a) == string(b) {
				continue
			}
		}
		put(f.Name, after[f.Name])
	}
}

// Seed stamps every shared record that has never been stamped, such as
// those from before this workspace was hosted in more than one place, so
// the first sync carries them.
func (s *Store) Seed() error {
	for _, t := range s.types.Types {
		if !s.shared(t.Name) {
			continue
		}
		recs, err := s.List(t.Name, ListOptions{})
		if err != nil {
			return err
		}
		for _, r := range recs {
			var n int
			s.db.QueryRow(`SELECT COUNT(*) FROM _state WHERE type = ? AND id = ?`, t.Name, r.ID).Scan(&n)
			if n == 0 {
				s.stamp(t, r.ID, nil, r.Fields, r.CreatedAt)
			}
		}
	}
	return nil
}

// Seen is the latest stamp this copy holds from each computer: what a
// peer need not send again.
func (s *Store) Seen() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT origin, clock FROM _seen`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var o, c string
		if err := rows.Scan(&o, &c); err != nil {
			return nil, err
		}
		out[o] = c
	}
	return out, rows.Err()
}

// Since is every stamp this copy holds that a peer who has seen `seen`
// has not: the fields whose winning write is newer than the latest the
// peer holds from the computer that made it.
func (s *Store) Since(seen map[string]string) ([]Stamp, error) {
	rows, err := s.db.Query(`SELECT type, id, field, value, clock FROM _state ORDER BY clock`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Stamp
	for rows.Next() {
		var st Stamp
		var v string
		if err := rows.Scan(&st.Type, &st.ID, &st.Field, &v, &st.Clock); err != nil {
			return nil, err
		}
		if st.Clock > seen[originOf(st.Clock)] {
			st.Value = json.RawMessage(v)
			out = append(out, st)
		}
	}
	return out, rows.Err()
}

// Apply takes stamps from another copy. Each wins where it is later than
// what this copy holds for that field, and the records they touched are
// written again from their fields. It says how many records changed.
func (s *Store) Apply(stamps []Stamp) (int, error) {
	touched := map[[2]string]bool{}
	for _, st := range stamps {
		s.clock.hear(st.Clock)
		var have string
		s.db.QueryRow(`SELECT clock FROM _state WHERE type = ? AND id = ? AND field = ?`, st.Type, st.ID, st.Field).Scan(&have)
		if st.Clock > have {
			if _, err := s.db.Exec(`INSERT OR REPLACE INTO _state (type, id, field, value, clock) VALUES (?, ?, ?, ?, ?)`, st.Type, st.ID, st.Field, string(st.Value), st.Clock); err != nil {
				return 0, err
			}
			touched[[2]string{st.Type, st.ID}] = true
		}
		o := originOf(st.Clock)
		var seen string
		s.db.QueryRow(`SELECT clock FROM _seen WHERE origin = ?`, o).Scan(&seen)
		if st.Clock > seen {
			s.db.Exec(`INSERT OR REPLACE INTO _seen (origin, clock) VALUES (?, ?)`, o, st.Clock)
		}
	}
	for k := range touched {
		if err := s.materialize(k[0], k[1]); err != nil {
			return 0, err
		}
	}
	return len(touched), nil
}

// materialize writes one record's row from its fields in _state, or
// removes it when it is deleted. A type this copy's schema does not know
// yet is kept in _state and written once it does.
func (s *Store) materialize(typeName, id string) error {
	t, ok := s.types.Get(typeName)
	if !ok || !s.shared(typeName) {
		return nil
	}
	rows, err := s.db.Query(`SELECT field, value FROM _state WHERE type = ? AND id = ?`, typeName, id)
	if err != nil {
		return err
	}
	fields := map[string]any{}
	deleted, created := false, time.Now().UTC()
	for rows.Next() {
		var f, v string
		if err := rows.Scan(&f, &v); err != nil {
			rows.Close()
			return err
		}
		var val any
		json.Unmarshal([]byte(v), &val)
		switch f {
		case fieldDeleted:
			deleted, _ = val.(bool)
		case fieldCreated:
			if c, _ := val.(string); c != "" {
				created, _ = time.Parse(time.RFC3339Nano, c)
			}
		default:
			fields[f] = val
		}
	}
	rows.Close()
	if deleted {
		s.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", quote(t.Name)), id)
		if s.AfterWrite != nil {
			s.AfterWrite(t.Name, id, nil)
		}
		return nil
	}
	clean, err := t.Normalize(fields)
	if err != nil {
		// A record another copy holds but this one's schema refuses (a
		// field required here and not there) waits in _state.
		return nil
	}
	_, err = s.put(t, id, clean, created, time.Now().UTC())
	return err
}
