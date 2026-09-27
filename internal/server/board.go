package server

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// boardField is the pick-list field whose choices are a board's columns:
// the one named in by, else the type's first shown pick-list.
func boardField(t *schema.Type, by any) (*schema.Field, error) {
	name, _ := by.(string)
	var enums []string
	for _, f := range t.Shown() {
		if f.Type != "enum" {
			continue
		}
		if name == "" || f.Name == name {
			f := f
			return &f, nil
		}
		enums = append(enums, f.Name)
	}
	if len(enums) == 0 {
		return nil, fmt.Errorf("a board needs a pick-list field for its columns, and %s has none; show it as a list, table or cards instead", t.Name)
	}
	return nil, fmt.Errorf("%s has no pick-list field %q to make the board's columns; it has %s", t.Name, name, strings.Join(enums, ", "))
}

// boardGroups puts each item in the column of its record's choice, in the
// order the field lists its choices, every choice a column even when empty
// so the board keeps its shape, then a column for those with no choice.
func boardGroups(f schema.Field, recs []*store.Record, items []any) []any {
	byValue := map[string][]any{}
	var none []any
	for i, rec := range recs {
		v, _ := rec.Fields[f.Name].(string)
		if v == "" {
			none = append(none, items[i])
			continue
		}
		byValue[v] = append(byValue[v], items[i])
	}
	groups := make([]any, 0, len(f.Values)+1)
	for _, v := range f.Values {
		groups = append(groups, map[string]any{"label": f.ValueLabel(v), "items": orEmpty(byValue[v])})
		delete(byValue, v)
	}
	// A value the field no longer lists still has its records shown.
	for _, rec := range recs {
		v, _ := rec.Fields[f.Name].(string)
		if list, ok := byValue[v]; ok {
			groups = append(groups, map[string]any{"label": f.ValueLabel(v), "items": list})
			delete(byValue, v)
		}
	}
	if len(none) > 0 {
		groups = append(groups, map[string]any{"label": "No " + strings.ToLower(fieldLabel(f)), "items": none})
	}
	return groups
}

func orEmpty(v []any) []any {
	if v == nil {
		return []any{}
	}
	return v
}
