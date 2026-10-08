package blocks

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// ChartComponent draws numbers from records: how many, or the sum of a
// field, grouped by a field or by the day, week or month of a date. The
// series is read when the page renders, so the picture is what is true
// now; without a type, the series is whatever the block carries.
const ChartComponent = "chart"

// resolveChart fills a chart block's series from the store when it names
// a type, and says in words what is wrong when something is.
func resolveChart(w *Workspace, props map[string]any, _ Place) map[string]any {
	out := copyProps(props)
	typeName, _ := props["type"].(string)
	if typeName == "" {
		return out
	}
	t, ok := w.Store.Types().Get(typeName)
	if !ok {
		out["problem"] = w.NoType(typeName)
		return out
	}
	g, problem := groupingOf(t, props)
	if problem != "" {
		out["problem"] = problem
		return out
	}
	recs, err := query.Filter(w.Store, t, Strs(props["where"]), "", 0, time.Now())
	if err != nil {
		out["problem"] = err.Error()
		return out
	}
	out["series"] = g.series(w.Store, t, recs)
	g.label(out, t)
	return out
}

// grouping is how a chart from records groups them: by a field, or by
// the day, week or month of a date (kind datetime), counting them or
// summing a number field.
type grouping struct {
	by, period, sum, kind string
	field                 *schema.Field
}

// groupingOf reads a chart's grouping from its props, or why it cannot.
func groupingOf(t *schema.Type, props map[string]any) (grouping, string) {
	var g grouping
	g.by, _ = props["by"].(string)
	g.period, _ = props["period"].(string)
	g.sum, _ = props["sum"].(string)
	if g.by == "" {
		return g, fmt.Sprintf("a chart from records needs by: the field to group by, or a date field with period day, week or month; %s has %s", t.Name, strings.Join(FieldsOfKind(t), ", "))
	}
	field, ok := t.Field(g.by)
	if !ok && g.by != "created_at" && g.by != "updated_at" {
		return g, fmt.Sprintf("%s has no field %q to group by; it has %s, and created_at and updated_at", t.Name, g.by, strings.Join(FieldsOfKind(t), ", "))
	}
	if g.sum != "" {
		if f, ok := t.Field(g.sum); !ok || (f.Type != "int" && f.Type != "float") {
			return g, noNumber(t, g.sum)
		}
	}
	g.field, g.kind = field, "datetime"
	if ok {
		g.kind = field.Type
	}
	if g.kind == "datetime" && g.period == "" {
		g.period = "month"
	}
	return g, ""
}

// series is the records counted or summed into their groups, in the
// order a person expects them.
func (g grouping) series(st *store.Store, t *schema.Type, recs []*store.Record) []any {
	totals := map[string]float64{}
	var keys []string
	for _, rec := range recs {
		key := bucket(st, t, g.field, g.kind, rec, g.by, g.period)
		if key == "" {
			continue
		}
		if _, seen := totals[key]; !seen {
			keys = append(keys, key)
		}
		if g.sum != "" {
			n, _ := rec.Fields[g.sum].(float64)
			if i64, ok := rec.Fields[g.sum].(int64); ok {
				n = float64(i64)
			}
			totals[key] += n
		} else {
			totals[key]++
		}
	}
	sortKeys(keys, totals, g.kind, g.field)
	series := make([]any, 0, len(keys))
	for _, k := range keys {
		series = append(series, map[string]any{"label": k, "value": totals[k]})
	}
	return series
}

// label names what the chart draws, unless its caption is given, and
// heads the numbers table's columns with what they are.
func (g grouping) label(out map[string]any, t *schema.Type) {
	if _, has := out["caption"]; !has {
		what := "How many " + schema.Plural(t.Name)
		if g.sum != "" {
			what = t.FieldDisplay(g.sum) + " of " + schema.Plural(t.Name)
		}
		out["caption"] = what + " by " + t.FieldWords(g.by)
	}
	out["groupLabel"] = t.FieldDisplay(g.by)
	if g.period != "" && g.kind == "datetime" {
		out["groupLabel"] = Capitalize(g.period)
	}
	out["valueLabel"] = Capitalize(schema.Plural(t.Name))
	if g.sum != "" {
		out["valueLabel"] = t.FieldDisplay(g.sum)
	}
}

// noNumber says why a chart cannot sum a field: it is not a number, and
// which are, or that leaving sum out counts instead.
func noNumber(t *schema.Type, sum string) string {
	out := fmt.Sprintf("sum needs a number field on %s; %q is not one", t.Name, sum)
	if nums := FieldsOfKind(t, "int", "float"); len(nums) > 0 {
		return out + "; its number fields are " + strings.Join(nums, ", ")
	}
	return out + ", and it has none; leave sum out to count " + schema.Plural(t.Name) + " instead"
}

// bucket is the group a record falls in: a field's value as a person
// reads it, or the day, week or month of a date.
func bucket(st *store.Store, t *schema.Type, f *schema.Field, kind string, rec *store.Record, by, period string) string {
	if kind == "datetime" {
		var v string
		switch by {
		case "created_at":
			v = rec.CreatedAt.UTC().Format(time.RFC3339)
		case "updated_at":
			v = rec.UpdatedAt.UTC().Format(time.RFC3339)
		default:
			v, _ = rec.Fields[by].(string)
		}
		ts, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return ""
		}
		local := ts.Local()
		// Labels a person reads under a bar: short, and in order when
		// sorted, since the groups are sorted by label.
		switch period {
		case "day":
			return local.Format("2006-01-02")
		case "week":
			monday := local.AddDate(0, 0, -((int(local.Weekday()) + 6) % 7))
			return monday.Format("2006-01-02")
		}
		return local.Format("2006-01")
	}
	if f == nil {
		return ""
	}
	v := Display(*f, rec.Fields[by])
	if f.Type == "ref" {
		v = records.RefTitle(st, *f, v)
	}
	if v == "" {
		return "None"
	}
	return v
}

// sortKeys orders the groups the way a person expects: dates in order,
// an enum in its declared order, anything else by size.
func sortKeys(keys []string, totals map[string]float64, kind string, f *schema.Field) {
	switch {
	case kind == "datetime":
		sort.Strings(keys)
	case kind == "enum" && f != nil:
		rank := map[string]int{}
		for i, v := range f.Values {
			rank[f.ValueLabel(v)] = i // the groups are labels, as bucket reads them
		}
		sort.SliceStable(keys, func(i, j int) bool { return rank[keys[i]] < rank[keys[j]] })
	default:
		sort.SliceStable(keys, func(i, j int) bool {
			if totals[keys[i]] != totals[keys[j]] {
				return totals[keys[i]] > totals[keys[j]]
			}
			return keys[i] < keys[j]
		})
	}
}
