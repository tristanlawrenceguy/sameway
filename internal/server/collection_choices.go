package server

import (
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A person's own choices on a collection: a few plain ways to narrow it
// and to sort it, where it is, without asking the assistant.
//
// The assistant's where is the base. A choice only adds conditions to it,
// so a person narrows what the block was built to show and never sees past
// it; the order is theirs to change. The choices live in the page's
// address, named after the block (c-<block>-sort, c-<block>-done), so two
// collections on one canvas keep their own, a live refresh keeps them, a
// link to the page keeps them, and a GET form with a button makes them
// with no script.

// collectionPlace is where a collection is being shown: the page its form
// comes back to, that page's address as asked for, and whether it is the
// block's own page. Nil where there is no page to come back to, as in a
// side pane or a block arriving live.
type collectionPlace struct {
	Path  string
	Query url.Values
	Own   bool
}

// onCanvas is the place of a collection block on a canvas: a person
// narrows and sorts it where it has room, on the canvas itself at full
// size, not in a pane or a strip.
func onCanvas(b *store.Record, convo *conversation) *collectionPlace {
	if convo == nil || str(b.Fields["region"], "main") != "main" || str(b.Fields["size"], "full") != "full" {
		return nil
	}
	return &collectionPlace{Path: convo.Path, Query: convo.Query}
}

// choice is one control: the sort, or one field to narrow by.
type choice struct {
	param, label string
	options      []option
}

// option is one thing a person can pick: what it is called, and the
// conditions it adds or the order it asks for.
type option struct {
	value, label string
	where        []string
	order        string
}

// fewEnough is how many a collection holds before its choices are worth
// the room they take on a canvas.
const fewEnough = 5

// collectionChoices derives the controls a type's fields make sense for:
// the sort, and at most one pick-list, one yes-or-no and one date field to
// narrow by. A field the base already fixes (done=false) is not offered,
// nor is a board's own column field.
func collectionChoices(t *schema.Type, where []string, order, by string) []choice {
	fixed := map[string]bool{by: by != ""}
	for _, w := range where {
		if c, err := query.Parse(t, w); err == nil && c.Op == "=" && c.Value != "" {
			fixed[c.Field] = true
		}
	}
	sort := choice{param: "sort", label: "Sort", options: []option{
		{value: "-created_at", label: "Newest first", order: "-created_at"},
		{value: "created_at", label: "Oldest first", order: "created_at"},
	}}
	if f, ok := t.Field(t.Title); ok && f.Type == "string" {
		sort.options = append(sort.options, option{value: f.Name, label: "A to Z", order: f.Name})
	}
	var enum, yes, day *schema.Field
	for _, f := range t.Shown() {
		f := f
		switch {
		case f.Type == "enum" && enum == nil && !fixed[f.Name] && len(f.Values) > 0:
			enum = &f
		case f.Type == "bool" && yes == nil && !fixed[f.Name]:
			yes = &f
		case (f.Type == "date" || f.Type == "datetime") && day == nil:
			day = &f
		}
	}
	if day != nil {
		sort.options = append(sort.options, option{value: day.Name, label: fieldLabel(*day) + " soonest first", order: day.Name})
	}
	if base := baseOrder(order); !hasOption(sort.options, base) {
		sort.options = append([]option{{value: base, label: "As set up", order: base}}, sort.options...)
	}
	out := []choice{sort}
	if enum != nil {
		c := choice{param: enum.Name, label: fieldLabel(*enum), options: []option{{label: "Any"}}}
		for _, v := range enum.Values {
			c.options = append(c.options, option{value: v, label: enum.ValueLabel(v), where: []string{enum.Name + "=" + v}})
		}
		out = append(out, c)
	}
	if yes != nil {
		c := choice{param: yes.Name, label: fieldLabel(*yes), options: []option{{label: "All"}}}
		for _, v := range []string{"false", "true"} {
			w := yes.Name + "=" + v
			c.options = append(c.options, option{value: v, label: capitalize(query.Words(t, []string{w})), where: []string{w}})
		}
		out = append(out, c)
	}
	if day != nil && !fixed[day.Name] {
		name := fieldLabel(*day)
		out = append(out, choice{param: day.Name, label: name, options: []option{
			{label: "Any time"},
			{value: "past", label: name + " before today", where: []string{day.Name + "<today"}},
			{value: "week", label: name + " in the next 7 days", where: []string{day.Name + ">=today", day.Name + "<=+7d"}},
		}})
	}
	return out
}

// baseOrder is the order a collection was set up with, newest first when
// none was given, as the store lists them.
func baseOrder(order string) string {
	if order == "" {
		return "-created_at"
	}
	return order
}

func hasOption(opts []option, value string) bool {
	for _, o := range opts {
		if o.value == value {
			return true
		}
	}
	return false
}

// applyChoices reads what a person picked on one collection from the
// page's address, fills the collection's choices (the form, the picks in
// words, the way back to how it was set up) and returns the where and
// order to query with and whether anything was picked. Only values the
// controls offer count, so the address cannot add a condition of its own.
func applyChoices(out map[string]any, choices []choice, at *collectionPlace, block string, where []string, order string) ([]string, string, bool) {
	prefix := "c-" + block + "-"
	id, _ := out["id"].(string)
	base := baseOrder(order)
	order = base
	var said, sortSaid []string
	var selects []any
	for _, c := range choices {
		none := ""
		if c.param == "sort" {
			none = base
		}
		picked := at.Query.Get(prefix + c.param)
		if !hasOption(c.options, picked) {
			picked = none
		}
		opts := make([]any, 0, len(c.options))
		for _, o := range c.options {
			opts = append(opts, map[string]any{"value": o.value, "label": o.label, "selected": o.value == picked})
			switch {
			case o.value != picked:
			case c.param == "sort":
				order = o.order
				if picked != none {
					sortSaid = []string{lowerFirst(o.label)}
				}
			case picked != none:
				where = append(append([]string{}, where...), o.where...)
				said = append(said, lowerFirst(o.label))
			}
		}
		selects = append(selects, map[string]any{
			"id": id + "-" + c.param, "name": prefix + c.param, "label": c.label, "options": opts,
		})
	}
	// What narrows it is said first, then the order.
	said = append(said, sortSaid...)
	active := len(said) > 0
	keep := url.Values{}
	for name, vals := range at.Query {
		if !strings.HasPrefix(name, prefix) {
			keep[name] = vals
		}
	}
	hidden := []any{}
	for _, name := range slices.Sorted(maps.Keys(keep)) {
		for _, v := range keep[name] {
			hidden = append(hidden, map[string]any{"name": name, "value": v})
		}
	}
	// The filters component draws them: a form, since a sort and up to
	// three fields are chosen together and applied at once.
	name := str(out["label"], str(out["type"], ""))
	ch := map[string]any{
		"shape": "form", "label": "Show and sort: " + name, "hideLabel": true, "context": name,
		"action": at.Path + "#" + id, "choices": selects, "keep": hidden,
	}
	if active {
		reset := at.Path
		if len(keep) > 0 {
			reset += "?" + keep.Encode()
		}
		ch["showing"] = strings.Join(said, ", ")
		ch["reset"] = reset + "#" + id
	}
	out["choices"] = ch
	return where, order, active
}

// lowerFirst puts a label mid-sentence: Not done becomes not done.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// offered says whether a collection shows its choices. controls: false
// turns them off and true on; left out, they are on where they help: on
// the block's own page, and on a canvas when more than a few match and the
// block was not set up as a short list. Never where there is no page to
// come back to.
func offered(props map[string]any, at *collectionPlace, block string, limit, matched int) bool {
	if at == nil || at.Path == "" || block == "" {
		return false
	}
	if on, ok := props["controls"].(bool); ok {
		return on
	}
	if at.Own {
		return true
	}
	prefix := "c-" + block + "-"
	for name := range at.Query {
		if strings.HasPrefix(name, prefix) {
			return true // a choice already made keeps its way back
		}
	}
	return limit > fewEnough && matched > fewEnough
}

// countWords says how many match: 12 tasks, 1 task.
func countWords(t *schema.Type, n int) string {
	if n == 0 {
		return ""
	}
	name := schema.Words(t.Name)
	if n != 1 {
		name = plural(t.Name)
	}
	return strconv.Itoa(n) + " " + name
}
