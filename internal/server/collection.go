package server

import (
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// collectionComponent is the block that shows the records matching a
// query. A block of it holds the query; the records are read when the
// page renders, so the list is always what is true now.
const collectionComponent = "collection"

// resolveCollection fills a collection block's props from the store: the
// matching records as items, the list page with the same query, and in
// words what is wrong when a condition is. block is the block's own id,
// which names the list's heading when it has no id of its own, so two
// lists of one type on a page are each named by their own heading.
func (s *Server) resolveCollection(props map[string]any, block string) map[string]any {
	return s.resolveCollectionAt(props, block, nil)
}

// resolveCollectionAt is resolveCollection on a page a person can narrow
// and sort it on: at says which page, and its address holds their
// choices (collection_choices.go).
func (s *Server) resolveCollectionAt(props map[string]any, block string, at *collectionPlace) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	delete(out, "choices") // the server's to fill, never the block's
	if id, _ := props["id"].(string); id == "" && block != "" {
		out["id"] = "collection-" + block
	}
	typeName, _ := props["type"].(string)
	t, ok := s.app.Types.Get(typeName)
	if !ok {
		out["problem"] = s.noType(typeName)
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
	// All of them, to know how many and whether there are more than are
	// shown: a list cut short says so, not only the whole list's page.
	now := time.Now()
	recs, err := query.Filter(s.app.Store, t, where, order, 0, now)
	if err != nil {
		out["problem"] = err.Error()
		return out
	}
	if offered(props, at, block, limit, len(recs)) {
		by := ""
		if f, err := boardField(t, props["by"]); err == nil && props["as"] == "board" {
			by = f.Name
		}
		choices := collectionChoices(t, where, order, by)
		if w, o, active := applyChoices(out, choices, at, block, where, order); active {
			if recs, err = query.Filter(s.app.Store, t, w, o, 0, now); err != nil {
				out["problem"] = err.Error()
				return out
			}
			where, order = w, o
		}
		out["choices"].(map[string]any)["count"] = countWords(t, len(recs))
	}
	if len(recs) > limit {
		recs, out["more"] = recs[:limit], true
	}
	full := props["detail"] == "full"
	show := strs(props["show"])
	columns := []any{}
	for _, name := range show {
		if f, ok := t.Field(name); ok {
			columns = append(columns, fieldLabel(*f))
		}
	}
	out["columns"] = columns
	out["titleLabel"] = label(t.Title)
	var by *schema.Field
	if props["as"] == "board" {
		if by, err = boardField(t, props["by"]); err != nil {
			out["problem"] = err.Error()
			return out
		}
	}
	items := make([]any, 0, len(recs))
	for _, rec := range recs {
		item := map[string]any{"title": s.title(t, rec), "href": "/t/" + t.Name + "/" + rec.ID}
		if len(show) > 0 {
			item["fields"] = s.fieldsOf(t, rec, show)
		} else if meta := metaOf(t, rec); meta != "" && by == nil {
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
	if by != nil {
		board, _ := out["id"].(string)
		if board == "" {
			board = "collection-" + t.Name
		}
		for i, rec := range recs {
			s.addMove(items[i].(map[string]any), t, *by, rec, board)
		}
		out["groups"] = boardGroups(*by, recs, items)
	}
	out["summary"] = query.Words(t, where)
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
	for _, f := range t.Shown() {
		if f.Type == "datetime" {
			if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
				return label(f.Name) + " " + when.Text(v)
			}
		}
	}
	for _, f := range t.Shown() {
		if f.Type == "enum" {
			if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
				return f.ValueLabel(v)
			}
		}
	}
	return ""
}

// textOf is the record's main text, for a full collection.
func textOf(t *schema.Type, rec *store.Record) string {
	for _, f := range t.Shown() {
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
func orderWords(t *schema.Type, order string) string {
	name := func(n string) string {
		if f, ok := t.Field(n); ok {
			return strings.ToLower(fieldLabel(*f))
		}
		return strings.ToLower(label(n))
	}
	switch {
	case order == "":
		return ""
	case strings.HasPrefix(order, "-"):
		return ", " + name(strings.TrimPrefix(order, "-")) + " largest or newest first"
	}
	return ", by " + name(order)
}

// fieldsOf is the chosen fields of a record as label and value, a ref by
// the title it points at and leading to it, in the order asked for, named
// as the record's own page names them.
func (s *Server) fieldsOf(t *schema.Type, rec *store.Record, names []string) []any {
	out := make([]any, 0, len(names))
	for _, name := range names {
		f, ok := t.Field(name)
		if !ok {
			continue
		}
		v := display(*f, rec.Fields[name])
		item := map[string]any{"label": fieldLabel(*f), "value": v}
		if f.Type == "ref" && v != "" {
			if _, err := s.app.Store.Get(f.To, v); err == nil {
				item["href"] = "/t/" + f.To + "/" + v
			}
			item["value"] = s.refTitle(*f, v)
		}
		out = append(out, item)
	}
	return out
}
