package server

import (
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// collectionComponent is the block that shows the records matching a
// query. A block of it holds the query; the records are read when the
// page renders, so the list is always what is true now.
const collectionComponent = "collection"

// resolveCollection fills a collection block's props from the store: the
// matching records as items, the list page with the same query, and in
// words what is wrong when a condition is.
func (s *Server) resolveCollection(props map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	typeName, _ := props["type"].(string)
	t, ok := s.app.Types.Get(typeName)
	if !ok {
		out["problem"] = "there is no content type " + typeName + "; the workspace has " + strings.Join(s.app.Types.Names(), ", ")
		return out
	}
	where := strs(props["where"])
	order, _ := props["order"].(string)
	limit := 20
	if n, ok := props["limit"].(float64); ok && n > 0 {
		limit = int(n)
	} else if n, ok := props["limit"].(int); ok && n > 0 {
		limit = n
	}
	recs, err := query.Filter(s.app.Store, t, where, order, limit, time.Now())
	if err != nil {
		out["problem"] = err.Error()
		return out
	}
	full := props["detail"] == "full"
	show := strs(props["show"])
	columns := []any{}
	for _, name := range show {
		if _, ok := t.Field(name); ok {
			columns = append(columns, label(name))
		}
	}
	out["columns"] = columns
	out["titleLabel"] = label(t.Title)
	items := make([]any, 0, len(recs))
	for _, rec := range recs {
		item := map[string]any{"title": titleOf(t, rec), "href": "/t/" + t.Name + "/" + rec.ID}
		if len(show) > 0 {
			item["fields"] = s.fieldsOf(t, rec, show)
		} else if meta := metaOf(t, rec); meta != "" {
			item["meta"] = meta
		}
		if full {
			if text := textOf(t, rec); text != "" {
				item["text"] = text
			}
		}
		if actions := markActions(t, rec); actions != nil {
			item["actions"] = actions
			out["pressable"] = true
		}
		items = append(items, item)
	}
	out["items"] = items
	out["summary"] = strings.Join(where, ", ")
	out["all"] = listPath(t.Name, where, order)
	return out
}

// listPath is the list page showing the same query.
func listPath(typeName string, where []string, order string) string {
	q := url.Values{}
	for _, w := range where {
		q.Add("where", w)
	}
	if order != "" {
		q.Set("order", order)
	}
	if len(q) == 0 {
		return "/t/" + typeName
	}
	return "/t/" + typeName + "?" + q.Encode()
}

// metaOf is the one thing worth saying beside a title in a list: the day
// it is due, or its state.
func metaOf(t *schema.Type, rec *store.Record) string {
	for _, f := range t.Fields {
		if f.Type == "datetime" {
			if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
				if ts, err := time.Parse(time.RFC3339, v); err == nil {
					return label(f.Name) + " " + ts.Local().Format("2006-01-02")
				}
			}
		}
	}
	for _, f := range t.Fields {
		if f.Type == "enum" {
			if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
				return capitalize(v)
			}
		}
	}
	return ""
}

// textOf is the record's main text, for a full collection.
func textOf(t *schema.Type, rec *store.Record) string {
	for _, f := range t.Fields {
		if f.Name != t.Title && isText(f) {
			if v := display(f, rec.Fields[f.Name]); v != "" {
				return v
			}
		}
	}
	return ""
}

// strs reads a list of strings out of props, whatever JSON made of it.
func strs(v any) []string {
	var out []string
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		for _, it := range x {
			if s, ok := it.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
	case string:
		if strings.TrimSpace(x) != "" {
			out = append(out, x)
		}
	}
	return out
}

// orderWords says an order in words for a list page's line.
func orderWords(order string) string {
	switch {
	case order == "":
		return ""
	case strings.HasPrefix(order, "-"):
		return ", " + strings.TrimPrefix(order, "-") + " largest or newest first"
	}
	return ", by " + order
}

// fieldsOf is the chosen fields of a record as label and value, a ref by
// the title it points at, in the order asked for.
func (s *Server) fieldsOf(t *schema.Type, rec *store.Record, names []string) []any {
	out := make([]any, 0, len(names))
	for _, name := range names {
		f, ok := t.Field(name)
		if !ok {
			continue
		}
		v := display(*f, rec.Fields[name])
		if f.Type == "ref" {
			v = s.refTitle(*f, v)
		}
		if f.Type == "datetime" && v != "" {
			if ts, err := time.Parse(time.RFC3339, v); err == nil {
				v = ts.Local().Format("2006-01-02")
			}
		}
		out = append(out, map[string]any{"label": label(name), "value": v})
	}
	return out
}
