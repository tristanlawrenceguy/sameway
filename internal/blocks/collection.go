package blocks

import (
	"net/url"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// CollectionComponent is the block that shows the records matching a
// query. A block of it holds the query; the records are read when the
// page renders, so the list is always what is true now.
const CollectionComponent = "collection"

// resolveCollection fills a collection block's props from the store: the
// matching records as items, the list page with the same query, and in
// words what is wrong when a condition is. The block's own id names the
// list's heading when it has no id of its own, so two lists of one type on
// a page are each named by their own heading. On a page a person can
// narrow and sort it on, the page's address holds their choices
// (collection_choices.go).
func resolveCollection(ws *Workspace, props map[string]any, at Place) map[string]any {
	block := at.Block
	out := copyProps(props)
	delete(out, "choices") // the server's to fill, never the block's
	if id, _ := props["id"].(string); id == "" && block != "" {
		out["id"] = "collection-" + block
	}
	typeName, _ := props["type"].(string)
	t, ok := ws.Store.Types().Get(typeName)
	if !ok {
		out["problem"] = ws.NoType(typeName)
		return out
	}
	where := Strs(props["where"])
	order, _ := props["order"].(string)
	limit := 20
	if n, ok := props["limit"].(float64); ok && n > 0 {
		limit = int(n)
	} else if n, ok := props["limit"].(int); ok && n > 0 {
		limit = n
	}
	// All of them, to know how many and whether there are more than are
	// shown: a list cut short says so, not only the whole list's page.
	recs, err := query.Filter(ws.Store, t, where, order, 0, ws.now())
	if err == nil && offered(props, at.Page, block, limit, len(recs)) {
		recs, where, order, err = ws.narrowed(out, t, props, at, recs, where, order)
	}
	if err != nil {
		out["problem"] = err.Error()
		return out
	}
	if len(recs) > limit {
		recs, out["more"] = recs[:limit], true
	}
	columns := []any{}
	for _, name := range Strs(props["show"]) {
		if f, ok := t.Field(name); ok {
			columns = append(columns, f.Display())
		}
	}
	out["columns"] = columns
	out["titleLabel"] = t.FieldDisplay(t.Title)
	var by *schema.Field
	if props["as"] == "board" {
		if by, err = BoardField(t, props["by"]); err != nil {
			out["problem"] = err.Error()
			return out
		}
	}
	ws.collectionItems(out, t, recs, props, by)
	out["summary"] = query.Words(t, where)
	out["all"] = listPath(t.Name, where, order)
	return out
}

// narrowed fills a collection's choices, and gives the records, where
// and order with a person's choices on the page applied.
func (ws *Workspace) narrowed(out map[string]any, t *schema.Type, props map[string]any, at Place, recs []*store.Record, where []string, order string) ([]*store.Record, []string, string, error) {
	by := ""
	if f, err := BoardField(t, props["by"]); err == nil && props["as"] == "board" {
		by = f.Name
	}
	choices := collectionChoices(t, where, order, by)
	setup := setupWords(t, where, order, choices) // collection_keep.go
	if w, o, active := applyChoices(out, choices, at.Page, at.Block, where, order); active {
		var err error
		if recs, err = query.Filter(ws.Store, t, w, o, 0, ws.now()); err != nil {
			return nil, nil, "", err
		}
		where, order, setup = w, o, ""
	}
	out["choices"].(map[string]any)["count"] = countSaying(t, len(recs), setup)
	return recs, where, order, nil
}

// collectionItems is the records as the collection's items: each by its
// title, told apart from another of the same name, with the fields asked
// for or what it says at a glance, its text when the list is full, its
// tick, and on a board, its column and the way to move it.
func (ws *Workspace) collectionItems(out map[string]any, t *schema.Type, recs []*store.Record, props map[string]any, by *schema.Field) {
	full := props["detail"] == "full"
	show := Strs(props["show"])
	items := make([]any, 0, len(recs))
	told := ws.RecordsApart(t, recs)
	var in records.Counts // what is in each, counted once for the list (records/glance_count.go)
	if len(show) == 0 && by == nil {
		in = records.CountsOf(ws.Store, t, recs)
	}
	for _, rec := range recs {
		item := map[string]any{"title": ws.title(t, rec), "href": "/t/" + t.Name + "/" + rec.ID}
		if told[rec.ID] != "" {
			item["context"] = told[rec.ID]
		}
		if len(show) > 0 {
			item["fields"] = ws.fieldsOf(t, rec, show)
		} else if meta := records.GlanceText(ws.Store, t, rec, ws.now(), ws.H24(), in); meta != "" && by == nil {
			item["meta"] = meta
		}
		if full {
			if text := TextOf(t, rec); text != "" {
				item["text"] = text
			}
		}
		if actions := ws.MarkActions(t, rec); actions != nil {
			markApart(actions, told[rec.ID])
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
			ws.addMove(items[i].(map[string]any), t, *by, rec, board)
		}
		out["groups"] = boardGroups(*by, recs, items)
	}
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

// TextOf is the record's main text, for a full collection.
func TextOf(t *schema.Type, rec *store.Record) string {
	for _, f := range t.Shown() {
		if f.Name != t.Title && IsText(f) {
			if v := Display(f, rec.Fields[f.Name], false); v != "" {
				return v
			}
		}
	}
	return ""
}

// OrderWords says an order in words for a list page's line.
func OrderWords(t *schema.Type, order string) string {
	name := t.FieldWords
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
func (ws *Workspace) fieldsOf(t *schema.Type, rec *store.Record, names []string) []any {
	out := make([]any, 0, len(names))
	for _, name := range names {
		f, ok := t.Field(name)
		if !ok {
			continue
		}
		v := Display(*f, rec.Fields[name], ws.H24())
		item := map[string]any{"label": f.Display(), "value": v}
		if f.Type == "ref" && v != "" {
			if _, err := ws.Store.Get(f.To, v); err == nil {
				item["href"] = "/t/" + f.To + "/" + v
			}
			item["value"] = ws.refTitle(*f, v)
		}
		out = append(out, item)
	}
	return out
}

func IsText(f schema.Field) bool {
	return f.Type == "text" || f.Type == "markdown"
}
