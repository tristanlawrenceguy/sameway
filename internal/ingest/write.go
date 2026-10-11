package ingest

import (
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// What comes in is written through records like every other write, so
// what follows the workspace hears it once: a row imported is heard when
// the caller logs the import with its ops (records.Made), and what the
// import does not log (a person found for a row, a calendar kept in step)
// is heard as the system's when it is written (records.ApplyOps). Either
// way an automation waiting for a record added runs for it.

// imported writes one row's record and says its id; the caller logs it.
func imported(st *store.Store, typ string, fields map[string]any) (string, error) {
	done, err := (&records.Book{Store: st}).Apply(records.Op{Type: typ, After: fields})
	if err != nil {
		return "", err
	}
	return done[0].ID, nil
}

// kept writes what the import does not log, and answers with the record
// as it now is, nil when the op removed it.
func kept(st *store.Store, op records.Op) (*store.Record, error) {
	done, err := records.ApplyOps(st, op)
	if err != nil {
		return nil, err
	}
	if done[0].After == nil {
		return nil, nil
	}
	return &store.Record{ID: done[0].ID, Type: op.Type, Fields: done[0].After}, nil
}
