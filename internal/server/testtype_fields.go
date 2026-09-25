package server

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// hasNoRecordValues says whether a test_type record carries any real data
// beyond its title and head fields (bools, enums, first date). If every
// non-head field is at its type's zero value, the page should show schema
// definitions rather than an empty list.
func hasNoRecordValues(t *schema.Type, rec *store.Record) bool {
	head := headFields(t, rec)
	for _, f := range t.Fields {
		if f.Name == "title" && t.Title != "" {
			continue
		}
		if head[f.Name] {
			continue
		}
		v := rec.Fields[f.Name]
		if !isZeroValue(f, v) {
			return false
		}
	}
	return true
}

// isZeroValue reports whether a field's stored value equals its type's zero.
func isZeroValue(f schema.Field, v any) bool {
	switch f.Type {
	case "int":
		if n, ok := v.(int64); ok && n == 0 {
			return true
		}
		return false
	case "float":
		if x, ok := v.(float64); ok && x == 0 {
			return true
		}
		return false
	case "bool":
		b, _ := v.(bool)
		return !b // zero for bool is false
	case "list":
		l, ok := v.([]any)
		return ok && len(l) == 0
	case "json":
		return v == nil
	default:
		s, _ := v.(string)
		return s == ""
	}
}

// schemaFields returns items in the same shape the fields component expects,
// built from a type's field definitions rather than record values. Each item
// shows the field name and its type so a person visiting a test_type detail
// page can see what fields are available even when no custom values exist.
func (s *Server) schemaFields(t *schema.Type) []any {
	var items []any
	for _, f := range t.Fields {
		if f.Name == "title" && t.Title != "" {
			continue // title is already the h1 heading
		}
		item := map[string]any{
			"label": fieldLabel(f),
			"value": f.Type,
			"prop":  f.Name,
		}
		if f.Description != "" {
			item["description"] = f.Description
		}
		switch f.Type {
		case "enum":
			item["value"] = fmt.Sprintf("values: %s", strings.Join(f.Values, ", "))
		case "ref":
			item["value"] = fmt.Sprintf("ref -> %s", f.To)
		case "list":
			item["value"] = fmt.Sprintf("list of %s", f.Of)
		}
		items = append(items, item)
	}
	return items
}
