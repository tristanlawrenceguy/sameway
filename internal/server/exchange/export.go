package exchange

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/blocks"
	"github.com/tristanlawrenceguy/sameway/internal/convert"
	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Whatever comes in goes out again: any list as a spreadsheet, people as
// contacts, anything with a date as a calendar, taking the same query the
// list page shows, so what is on the page is what is in the file. A
// calendar's address is also a feed a calendar app can subscribe to.

// exportable says whether a type's records are a person's to take out:
// the system's own (the conversation, the log, the questions) are not.
func exportable(t *schema.Type) bool { return t.Content() }

// exportFile answers /export/<type>.<ext>?where=…&order=… with the file.
func (s *Service) exportFile(w http.ResponseWriter, r *http.Request) {
	name, ext, _ := strings.Cut(r.PathValue("file"), ".")
	t, ok := s.app.Types.Get(name)
	if !ok || !exportable(t) {
		http.NotFound(w, r)
		return
	}
	f, ok := export.ByExt(t, ext)
	if !ok {
		http.Error(w, fmt.Sprintf("%s cannot be taken out as .%s; it can be %s", schema.Plural(t.Name), ext, formatList(t)), http.StatusNotFound)
		return
	}
	recs, err := s.exportRecords(t, r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var buf bytes.Buffer
	if err := export.Write(&buf, f, t, recs, s.RefTitle); err != nil {
		s.Fail(w, err)
		return
	}
	w.Header().Set("Content-Type", f.Type)
	web.Attachment(w, fmt.Sprintf("%s %s.%s", blocks.Capitalize(schema.Plural(t.Name)), s.Now().Format("2006-01-02"), f.Ext))
	w.Write(buf.Bytes())
}

// exportRecords are the records a list page with this query shows.
func (s *Service) exportRecords(t *schema.Type, q url.Values) ([]*store.Record, error) {
	return query.Filter(s.app.Store, t, q["where"], q.Get("order"), 0, s.Now())
}

func formatList(t *schema.Type) string {
	var names []string
	for _, f := range export.For(t) {
		names = append(names, "."+f.Ext)
	}
	return strings.Join(names, ", ")
}

// ExportLinks is the list page's way out: each format the type can be,
// with the page's own query, each saying how big it is (see the export
// component). recs are the records the page matched, which the files hold.
func (s *Service) ExportLinks(t *schema.Type, q url.Values, recs []*store.Record) string {
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
	what := "these " + fmt.Sprint(len(recs)) + " " + schema.Plural(t.Name)
	if len(recs) == 1 {
		what = "this " + schema.Words(t.Name)
	}
	return string(s.Component("export", map[string]any{"what": what, "items": items}))
}

// exportSize is how big a file will be, by making it: a workspace's lists
// are small, and a size said is a size people can decide on.
func (s *Service) exportSize(f export.Format, t *schema.Type, recs []*store.Record) string {
	var n counter
	if err := export.Write(&n, f, t, recs, s.RefTitle); err != nil {
		return ""
	}
	return blocks.SizeWords(int64(n))
}

// counter counts what is written to it.
type counter int64

func (c *counter) Write(p []byte) (int, error) { *c += counter(len(p)); return len(p), nil }

// transcriptFile answers a recording's words as subtitles (.srt) or as
// plain text (.txt), from its text, so a correction goes out too.
func (s *Service) transcriptFile(w http.ResponseWriter, r *http.Request) {
	rec, err := s.app.Store.Get(records.FileType, r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cues := s.Media().Heard(rec)
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
	web.Attachment(w, title+"."+ext)
	w.Write([]byte(body))
}

func filepathExt(p string) string {
	if i := strings.LastIndex(p, "."); i >= 0 && !strings.Contains(p[i:], "/") {
		return p[i:]
	}
	return ""
}
