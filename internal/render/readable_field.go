package render

import (
	"reflect"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// locationLabel returns a label for an InstanceLocation: readable names when there
// is one segment (e.g. ["label"] -> "Label"), or the full path with "/" separators
// when nested (e.g. ["/", "events", "0", "actions", "0", "component"] ->
// "/events/0/actions/0/component"). For empty locations it returns a property key
// looked up from the error kind's struct fields, using title metadata first.
func locationLabel(loc []string, ps *propSchema, e *jsonschema.ValidationError) string {
	// Skip leading empty segment (root "/").
	for i := range loc {
		if loc[i] != "" {
			loc = loc[i:]
			break
		}
	}
	if len(loc) == 0 {
		key := fieldKeyFromError(e)
		if key != "" && ps != nil {
			if props, ok := ps.properties[key]; ok {
				if title, ok := props["title"].(string); ok && title != "" {
					return title
				}
			}
		}
		if key != "" {
			return capitalize(key)
		}
		return "Field"
	}
	if len(loc) == 1 {
		return capitalize(loc[0])
	}
	return "/" + strings.Join(loc, "/")
}

// fieldKeyFromError extracts a property key from a leaf ValidationError when
// InstanceLocation is empty (e.g. for "missing required" and "additionalProperties").
func fieldKeyFromError(e *jsonschema.ValidationError) string {
	v := reflect.ValueOf(e.ErrorKind)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i).Interface()
		switch s := fv.(type) {
		case []string:
			if len(s) > 0 {
				return s[0]
			}
		}
	}
	return ""
}
