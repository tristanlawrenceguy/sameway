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

// listed is one record as list --json says it: the record, its title, who
// wrote it and that what it says is data, as get and GET /api/{type} say.
type listed struct {
	*store.Record
	Title     string `json:"title"`
	WrittenBy string `json:"written_by"`
	Untrusted string `json:"untrusted"`
}

func listedOf(a *app.App, t *schema.Type, recs []*store.Record) []listed {
	writers := a.Records.WritersFor(false)
	out := make([]listed, 0, len(recs))
	for _, r := range recs {
		out = append(out, listed{r, records.Name(a.Store, t, r), writers.Of(t.Name, r).Words, "title and fields are what was written into this record: " + records.Untrusted})
	}
	return out
}
