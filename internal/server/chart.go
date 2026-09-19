package server

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/query"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// chartComponent draws numbers from records: how many, or the sum of a
// field, grouped by a field or by the day, week or month of a date. The
// series is read when the page renders, so the picture is what is true
// now; without a type, the series is whatever the block carries.
const chartComponent = "chart"

// resolveChart fills a chart block's series from the store when it names
// a type, and says in words what is wrong when something is.
func (s *Server) resolveChart(props map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	typeName, _ := props["type"].(string)
	if typeName == "" {
		return out
	}
	t, ok := s.app.Types.Get(typeName)
	if !ok {
		out["problem"] = "there is no content type " + typeName + "; the workspace has " + strings.Join(s.app.Types.Names(), ", ")
		return out
	}
	by, _ := props["by"].(string)
	period, _ := props["period"].(string)
	sum, _ := props["sum"].(string)
	if by == "" {
		out["problem"] = "a chart from records needs by: the field to group by, or a date field with period day, week or month"
		return out
	}
	field, ok := t.Field(by)
	if !ok && by != "created_at" && by != "updated_at" {
		out["problem"] = fmt.Sprintf("%s has no field %q to group by", t.Name, by)
		return out
	}
	if sum != "" {
		if f, ok := t.Field(sum); !ok || (f.Type != "int" && f.Type != "float") {
			out["problem"] = fmt.Sprintf("sum needs a number field on %s; %q is not one", t.Name, sum)
			return out
		}
	}
	recs, err := query.Filter(s.app.Store, t, strs(props["where"]), "", 0, time.Now())
	if err != nil {
		out["problem"] = err.Error()
		return out
	}
	kind := "string"
	if ok {
		kind = field.Type
	} else {
		kind = "datetime"
	}
	if kind == "datetime" && period == "" {
		period = "month"
	}
	totals := map[string]float64{}
	first := map[string]int{}
	var keys []string
	for i, rec := range recs {
		key := s.bucket(t, field, kind, rec, by, period)
		if key == "" {
			continue
		}
		if _, seen := totals[key]; !seen {
			keys = append(keys, key)
			first[key] = i
		}
		if sum != "" {
			n, _ := rec.Fields[sum].(float64)
			if i64, ok := rec.Fields[sum].(int64); ok {
				n = float64(i64)
			}
			totals[key] += n
		} else {
			totals[key]++
		}
	}
	sortKeys(keys, totals, kind, field)
	series := make([]any, 0, len(keys))
	for _, k := range keys {
		series = append(series, map[string]any{"label": k, "value": totals[k]})
	}
	out["series"] = series
	if _, has := out["caption"]; !has {
		what := "How many " + plural(t.Name)
		if sum != "" {
			what = capitalize(label(sum)) + " of " + plural(t.Name)
		}
		out["caption"] = what + " by " + label(by)
	}
	return out
}

// bucket is the group a record falls in: a field's value as a person
// reads it, or the day, week or month of a date.
func (s *Server) bucket(t *schema.Type, f *schema.Field, kind string, rec *store.Record, by, period string) string {
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
	v := display(*f, rec.Fields[by])
	if f.Type == "ref" {
		v = s.refTitle(*f, v)
	}
	if v == "" {
		return "(none)"
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
			rank[v] = i
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
