package server_test

import (
	"testing"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// written writes a record as every way in does, through records and
// logged as the person's, so what waits on a change (an automation,
// sorting) hears it; the store itself tells nobody. action is created,
// updated or deleted.
func written(t *testing.T, st *store.Store, action, typ, id string, fields map[string]any) *store.Record {
	t.Helper()
	rec, _, err := records.WriteAs(st, records.Who{Actor: "human"}, action, typ, id, fields)
	if err != nil {
		t.Fatal(err)
	}
	return rec
}
