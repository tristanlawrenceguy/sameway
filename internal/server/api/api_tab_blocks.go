package api

import (
	"net/http"
	"sort"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// One tab's blocks. Laying a tab out (POST /api/arrange) takes every block
// on it, and GET /api/block gave every block on every tab, header and
// footer among them: an agent filtered them by hand, left one out, and was
// refused (crew finding 0747). ?canvas=<id> keeps one tab's, in the order
// the page reads them; canvas= with nothing after it is Home.

// tabOnly is recs kept to the tab ?canvas= names, when the type is block
// and it names one; recs as they are otherwise.
func tabOnly(r *http.Request, recs []*store.Record) []*store.Record {
	q := r.URL.Query()
	if r.PathValue("type") != records.BlockType || !q.Has("canvas") {
		return recs
	}
	recs = records.OnCanvas(recs, q.Get("canvas"))
	if q.Get("order") == "" {
		sort.SliceStable(recs, func(i, j int) bool {
			a, _ := recs[i].Fields["position"].(int64)
			b, _ := recs[j].Fields["position"].(int64)
			return a < b
		})
	}
	return recs
}
