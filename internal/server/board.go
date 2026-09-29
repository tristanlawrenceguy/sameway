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

// addMove gives a board's card an id and the form that moves it: its
// column picked from the field's choices and a Move button named with the
// card, which posts the field like any edit, so it is logged and can be
// undone, and works with no script. The page comes back to the Move
// button, in the card's new column, so focus returns to what was pressed.
func (s *Server) addMove(item map[string]any, t *schema.Type, f schema.Field, rec *store.Record, board string) {
	at := board + "-" + rec.ID
	item["at"] = at
	options := make([]any, 0, len(f.Values))
	for _, v := range f.Values {
		options = append(options, map[string]any{"value": v, "label": f.ValueLabel(v)})
	}
	value, _ := rec.Fields[f.Name].(string)
	actions, _ := item["actions"].([]any)
	// Every card's choice is named after its card, as its Move is: a board
	// is a column of selects all called Status otherwise.
	title := withContext(s.title(t, rec), str(item["context"], ""))
	item["actions"] = append(actions, map[string]any{"component": "move", "props": map[string]any{
		"action": "/t/" + t.Name + "/" + rec.ID + "/props", "title": title, "id": at + "-go",
		"select": map[string]any{
			"id": at + "-move", "name": "prop-" + f.Name, "label": fieldLabel(f), "context": title,
			"as": "dropdown", "value": value, "options": options,
		},
	}})
}

func orEmpty(v []any) []any {
	if v == nil {
		return []any{}
	}
	return v
}
