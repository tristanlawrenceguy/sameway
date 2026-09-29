package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// An agent pays for every word it reads, in a window that runs out; a
// person reads a page at a time and skims the rest. Lists come a page at
// a time when asked (?page=, as search does), and any read can ask for
// only the fields it needs (?fields=title,status).

// listPage is the page a list asked for: how many a page holds and which
// page, or zero when it asked for all of them.
func listPage(q url.Values, limit int) (size, page int, err error) {
	p := q.Get("page")
	if p == "" {
		return 0, 0, nil
	}
	page, err = strconv.Atoi(p)
	if err != nil || page < 1 {
		return 0, 0, fmt.Errorf("page is a page number from 1, not %q", p)
	}
	size = limit
	if size <= 0 {
		size = 50
	}
	return size, page, nil
}

// onePage cuts recs to one page and says where it is: total, page, pages,
// and next, the address of the next page when there is one.
func onePage(r *http.Request, recs []*store.Record, size, page int) ([]*store.Record, map[string]any) {
	total := len(recs)
	pages := max((total+size-1)/size, 1)
	lo, hi := min((page-1)*size, total), min(page*size, total)
	about := map[string]any{"total": total, "page": page, "pages": pages}
	if page < pages {
		q := r.URL.Query()
		q.Set("page", strconv.Itoa(page+1))
		about["next"] = r.URL.Path + "?" + q.Encode()
	}
	return recs[lo:hi], about
}

// onlyFields is the fields a read asked for, checked against the type, or
// nil for all of them.
func (s *Server) onlyFields(r *http.Request, typ string) ([]string, error) {
	asked := r.URL.Query().Get("fields")
	if asked == "" {
		return nil, nil
	}
	t, ok := s.app.Types.Get(typ)
	if !ok {
		return nil, fmt.Errorf("no content type %q", typ)
	}
	var names, bad []string
	for _, n := range strings.Split(asked, ",") {
		if n = strings.TrimSpace(n); n == "" {
			continue
		}
		if _, ok := t.Field(n); !ok {
			bad = append(bad, n)
		}
		names = append(names, n)
	}
	if len(bad) > 0 {
		var have []string
		for _, f := range t.Fields {
			have = append(have, f.Name)
		}
		return nil, fmt.Errorf("%s has no %s; its fields are %s", t.Name, strings.Join(bad, ", "), strings.Join(have, ", "))
	}
	return names, nil
}

// trimmed is a record with only the named fields, or as it is for none.
func trimmed(rec *store.Record, names []string) *store.Record {
	if names == nil {
		return rec
	}
	out := *rec
	out.Fields = map[string]any{}
	for _, n := range names {
		if v, ok := rec.Fields[n]; ok {
			out.Fields[n] = v
		}
	}
	return &out
}
