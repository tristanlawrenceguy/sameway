package cli

import (
	"github.com/tristanlawrenceguy/sameway/internal/app"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// viewOf is a record as get --json says it: the one view of a record every
// way in gives (records/view.go), what GET /api/{type}/{id} and get_record
// answer, with who wrote it, its title, its page and what it connects to.
// The CLI gave the bare row, and an agent on it missed all of that.
func viewOf(a *app.App, t *schema.Type, rec *store.Record) records.RecordView {
	return a.Records.ViewOf(t, rec, a.Records.WritersFor(false))
}
