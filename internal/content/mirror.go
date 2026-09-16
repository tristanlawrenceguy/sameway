package content

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Report says what an export or an import did.
type Report struct {
	Written  int      `json:"written,omitempty"`
	Removed  int      `json:"removed,omitempty"`
	Created  int      `json:"created,omitempty"`
	Updated  int      `json:"updated,omitempty"`
	Deleted  int      `json:"deleted,omitempty"`
	Problems []string `json:"problems,omitempty"`
}

// Logger is told about each record an import changed, with the record as
// it now is (or was, when deleted) and what it was before, so the change
// lands in the activity log like any other and can be undone.
type Logger func(action string, rec *store.Record, before map[string]any)

// Export rewrites the folder from the database: every mirrored record gets
// its file, and a file with no record behind it goes. The hook keeps the
// folder current; this is for a workspace that predates it, or one whose
// database was changed behind its back.
func (m Mirror) Export(st *store.Store) (Report, error) {
	var r Report
	for _, t := range m.mirrored() {
		recs, err := st.List(t.Name, store.ListOptions{})
		if err != nil {
			return r, err
		}
		have := map[string]bool{}
		for _, rec := range recs {
			have[rec.ID] = true
			if err := m.write(rec); err != nil {
				return r, err
			}
			r.Written++
		}
		ids, err := m.ids(t.Name)
		if err != nil {
			return r, err
		}
		for _, id := range ids {
			if have[id] {
				continue
			}
			if err := m.remove(t.Name, id); err != nil {
				return r, err
			}
			r.Removed++
		}
	}
	return r, nil
}

// Import makes the database match the folder: a file with no record makes
// one, a file that differs from its record changes it, a record with no
// file goes. Each change is given to log with what was there before. A
// file that cannot be read is reported and skipped, never a reason to stop.
func (m Mirror) Import(st *store.Store, log Logger) (Report, error) {
	var r Report
	if log == nil {
		log = func(string, *store.Record, map[string]any) {}
	}
	for _, t := range m.mirrored() {
		ids, err := m.ids(t.Name)
		if err != nil {
			return r, err
		}
		inFiles := map[string]bool{}
		for _, id := range ids {
			inFiles[id] = true
			data, err := os.ReadFile(m.Path(t.Name, id))
			if err != nil {
				r.Problems = append(r.Problems, fmt.Sprintf("%s/%s.md: %v", t.Name, id, err))
				continue
			}
			fields, created, updated, err := Decode(t, data)
			if err != nil {
				r.Problems = append(r.Problems, fmt.Sprintf("%s/%s.md: %v", t.Name, id, err))
				continue
			}
			current, getErr := st.Get(t.Name, id)
			if getErr == nil {
				clean, err := t.Normalize(fields)
				if err == nil && same(clean, current.Fields) {
					continue
				}
			}
			rec, err := st.Put(t.Name, id, fields, created, updated)
			if err != nil {
				r.Problems = append(r.Problems, fmt.Sprintf("%s/%s.md: %v", t.Name, id, err))
				continue
			}
			if getErr == nil {
				r.Updated++
				log("updated", rec, current.Fields)
			} else {
				r.Created++
				log("created", rec, nil)
			}
		}
		recs, err := st.List(t.Name, store.ListOptions{})
		if err != nil {
			return r, err
		}
		for _, rec := range recs {
			if inFiles[rec.ID] {
				continue
			}
			if err := st.Delete(t.Name, rec.ID); err != nil {
				r.Problems = append(r.Problems, fmt.Sprintf("%s %s: %v", t.Name, rec.ID, err))
				continue
			}
			r.Deleted++
			log("deleted", rec, rec.Fields)
		}
	}
	return r, nil
}

func (m Mirror) mirrored() []*schema.Type {
	var out []*schema.Type
	if m.Types == nil {
		return nil
	}
	for _, t := range m.Types.Types {
		if m.Mirrored(t.Name) {
			out = append(out, t)
		}
	}
	return out
}

// same compares field maps as JSON, the way they are stored.
func same(a, b map[string]any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
