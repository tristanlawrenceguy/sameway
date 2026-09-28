package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Whatever comes in goes out again: any list as a spreadsheet, people as
// contacts, anything with a date as a calendar, taking the same query the
// list page shows, so what is on the page is what is in the file. A
// calendar's address is also a feed a calendar app can subscribe to.

// exportable says whether a type's records are a person's to take out:
// the system's own (the conversation, the log, the questions) are not.
func exportable(t *schema.Type) bool { return !t.Internal && !t.Hidden }

// exportFile answers /export/<type>.<ext>?where=…&order=… with the file.
func (s *Server) exportFile(w http.ResponseWriter, r *http.Request) {
	name, ext, _ := strings.Cut(r.PathValue("file"), ".")
	t, ok := s.app.Types.Get(name)
	if !ok || !exportable(t) {
		http.NotFound(w, r)
		return
	}
	f, ok := export.ByExt(t, ext)
	if !ok {
		http.Error(w, fmt.Sprintf("%s cannot be taken out as .%s; it can be %s", plural(t.Name), ext, formatList(t)), http.StatusNotFound)
		return
	}
	recs, err := s.exportRecords(t, r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var buf bytes.Buffer
	if err := export.Write(&buf, f, t, recs, s.exportTitles); err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", f.Type)
	attachment(w, fmt.Sprintf("%s %s.%s", capitalize(plural(t.Name)), time.Now().Format("2006-01-02"), f.Ext))
	w.Write(buf.Bytes())
}

// exportRecords are the records a list page with this query shows.
func (s *Server) exportRecords(t *schema.Type, q url.Values) ([]*store.Record, error) {
	where, order := q["where"], q.Get("order")
	if len(where) > 0 || order != "" {
		return query.Filter(s.app.Store, t, where, order, 0, time.Now())
	}
	return s.app.Store.List(t.Name, store.ListOptions{})
}

func (s *Server) exportTitles(f schema.Field, id string) string { return s.refTitle(f, id) }

func formatList(t *schema.Type) string {
	var names []string
	for _, f := range export.For(t) {
		names = append(names, "."+f.Ext)
	}
	return strings.Join(names, ", ")
}

// exportLinks is the list page's way out: each format the type can be,
// with the page's own query, each saying how big it is (see the export
// component). recs are the records the page matched, which the files hold.
func (s *Server) exportLinks(t *schema.Type, q url.Values, recs []*store.Record) string {
	if !exportable(t) || len(recs) == 0 {
		return ""
	}
	query := ""
	if keep := (url.Values{"where": q["where"], "order": {q.Get("order")}}); len(q["where"]) > 0 || q.Get("order") != "" {
		if keep.Get("order") == "" {
			keep.Del("order")
		}
		query = "?" + keep.Encode()
	}
	var items []any
	for _, f := range export.For(t) {
		items = append(items, map[string]any{"href": "/export/" + t.Name + "." + f.Ext + query, "format": f.Ext, "size": s.exportSize(f, t, recs)})
	}
	what := "these " + fmt.Sprint(len(recs)) + " " + plural(t.Name)
	if len(recs) == 1 {
		what = "this " + schema.Words(t.Name)
	}
	return string(s.component("export", map[string]any{"what": what, "items": items}))
}

// exportSize is how big a file will be, by making it: a workspace's lists
// are small, and a size said is a size people can decide on.
func (s *Server) exportSize(f export.Format, t *schema.Type, recs []*store.Record) string {
	var n counter
	if err := export.Write(&n, f, t, recs, s.exportTitles); err != nil {
		return ""
	}
	return sizeWords(int64(n))
}

// counter counts what is written to it.
type counter int64

func (c *counter) Write(p []byte) (int, error) { *c += counter(len(p)); return len(p), nil }

// attachment says a response is a file to keep, by its name: filename for
// every browser, and filename* for a name beyond ASCII (RFC 6266).
func attachment(w http.ResponseWriter, name string) {
	plain := strings.Map(func(r rune) rune {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' || r == '/' {
			return '_'
		}
		return r
	}, name)
	v := fmt.Sprintf(`attachment; filename="%s"`, plain)
	if plain != name {
		v += "; filename*=UTF-8''" + url.PathEscape(strings.ReplaceAll(name, "/", "_"))
	}
	w.Header().Set("Content-Disposition", v)
}

// transcriptFile answers a recording's words as subtitles (.srt) or as
// plain text (.txt), from its text, so a correction goes out too.
func (s *Server) transcriptFile(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(FileType, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cues := s.heard(rec)
	if len(cues) == 0 {
		http.NotFound(w, r)
		return
	}
	title, _ := rec.Fields["title"].(string)
	ext := strings.TrimPrefix(filepathExt(r.URL.Path), ".")
	var body string
	switch ext {
	case "srt":
		w.Header().Set("Content-Type", "application/x-subrip; charset=utf-8")
		body = convert.SRT(cues)
	case "txt":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		body = convert.Transcript(cues) + "\n"
	default:
		http.NotFound(w, r)
		return
	}
	attachment(w, title+"."+ext)
	w.Write([]byte(body))
}

func filepathExt(p string) string {
	if i := strings.LastIndex(p, "."); i >= 0 && !strings.Contains(p[i:], "/") {
		return p[i:]
	}
	return ""
}

// RefTitle is what a ref points at, by its title, for the command line's
// exports.
func (s *Server) RefTitle(f schema.Field, id string) string { return s.refTitle(f, id) }
