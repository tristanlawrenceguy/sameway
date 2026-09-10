package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ValidationError lists every problem with a record so a form can show them all.
type ValidationError struct {
	Problems map[string]string // field name -> message
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Problems))
	for k, v := range e.Problems {
		parts = append(parts, k+": "+v)
	}
	return "invalid record: " + strings.Join(parts, "; ")
}

// Normalize applies defaults, checks types and constraints, and returns a
// clean record that only contains known fields. Unknown fields are an error
// so typos never silently disappear.
func (t *Type) Normalize(in map[string]any) (map[string]any, error) {
	out := map[string]any{}
	problems := map[string]string{}
	for k := range in {
		if _, ok := t.Field(k); !ok {
			problems[k] = "unknown field"
		}
	}
	for _, f := range t.Fields {
		v, present := in[f.Name]
		if !present || v == nil || v == "" {
			if f.Default != nil {
				v = f.Default
			} else if f.Required {
				problems[f.Name] = "is required"
				continue
			} else {
				out[f.Name] = zero(f)
				continue
			}
		}
		clean, err := coerce(f, v)
		if err != nil {
			problems[f.Name] = err.Error()
			continue
		}
		out[f.Name] = clean
	}
	if len(problems) > 0 {
		return nil, &ValidationError{Problems: problems}
	}
	return out, nil
}

// DefaultValue is what a field holds when nothing was stored for it: its
// declared default, or the zero value for its type. Records written before
// a field was added read back through this, so adding a field to a schema
// needs no backfill and never surfaces a null the schema does not admit.
func DefaultValue(f Field) any {
	if f.Default != nil {
		if v, err := coerce(f, f.Default); err == nil {
			return v
		}
	}
	return zero(f)
}

func zero(f Field) any {
	switch f.Type {
	case "int":
		return int64(0)
	case "float":
		return float64(0)
	case "bool":
		return false
	case "list":
		return []any{}
	case "json":
		// Absent JSON is nil, not an empty object: the shape is the caller's.
		return nil
	default:
		return ""
	}
}

// coerce accepts values from JSON (float64, bool, []any), from forms and the
// CLI (strings), and from Go code, and returns the canonical Go value.
func coerce(f Field, v any) (any, error) {
	switch f.Type {
	case "string", "text", "markdown":
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("must be text")
		}
		if f.MaxLength > 0 && len([]rune(s)) > f.MaxLength {
			return nil, fmt.Errorf("must be at most %d characters", f.MaxLength)
		}
		return s, nil
	case "enum":
		s, ok := v.(string)
		if !ok || !contains(f.Values, s) {
			return nil, fmt.Errorf("must be one of %s", strings.Join(f.Values, ", "))
		}
		return s, nil
	case "int":
		switch n := v.(type) {
		case int:
			return int64(n), nil
		case int64:
			return n, nil
		case float64:
			if n != float64(int64(n)) {
				return nil, fmt.Errorf("must be a whole number")
			}
			return int64(n), nil
		case string:
			i, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("must be a whole number")
			}
			return i, nil
		}
		return nil, fmt.Errorf("must be a whole number")
	case "float":
		switch n := v.(type) {
		case float64:
			return n, nil
		case int:
			return float64(n), nil
		case int64:
			return float64(n), nil
		case string:
			x, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
			if err != nil {
				return nil, fmt.Errorf("must be a number")
			}
			return x, nil
		}
		return nil, fmt.Errorf("must be a number")
	case "bool":
		switch b := v.(type) {
		case bool:
			return b, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(b)) {
			case "true", "yes", "on", "1":
				return true, nil
			case "false", "no", "off", "0", "":
				return false, nil
			}
		}
		return nil, fmt.Errorf("must be true or false")
	case "datetime":
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("must be an RFC 3339 time")
		}
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("must be an RFC 3339 time like 2026-09-10T12:00:00Z")
		}
		return ts.UTC().Format(time.RFC3339), nil
	case "list":
		return coerceList(f, v)
	case "json":
		return coerceJSON(v)
	}
	return nil, fmt.Errorf("unsupported type %s", f.Type)
}

func coerceList(f Field, v any) (any, error) {
	var items []any
	switch l := v.(type) {
	case []any:
		items = l
	case []string:
		for _, s := range l {
			items = append(items, s)
		}
	case string:
		// Forms and the CLI send comma separated values, or a JSON array.
		if strings.HasPrefix(strings.TrimSpace(l), "[") {
			if err := json.Unmarshal([]byte(l), &items); err != nil {
				return nil, fmt.Errorf("must be a JSON array or comma separated values")
			}
		} else {
			// One per line when the person used lines, commas otherwise.
			sep := ","
			if strings.ContainsAny(l, "\r\n") {
				sep = "\n"
				l = strings.ReplaceAll(l, "\r\n", "\n")
			}
			for _, part := range strings.Split(l, sep) {
				if p := strings.TrimSpace(part); p != "" {
					items = append(items, p)
				}
			}
		}
	default:
		return nil, fmt.Errorf("must be a list")
	}
	elem := Field{Type: f.Of}
	if elem.Type == "" {
		elem.Type = "string"
	}
	out := make([]any, 0, len(items))
	for i, item := range items {
		c, err := coerce(elem, item)
		if err != nil {
			return nil, fmt.Errorf("item %d %v", i+1, err)
		}
		out = append(out, c)
	}
	return out, nil
}

func coerceJSON(v any) (any, error) {
	if s, ok := v.(string); ok {
		var parsed any
		if err := json.Unmarshal([]byte(s), &parsed); err != nil {
			return nil, fmt.Errorf("must be valid JSON")
		}
		return parsed, nil
	}
	// Round-trip so Go values (structs, typed slices) become the plain JSON
	// shapes every reader sees: maps, []any, float64.
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("must be valid JSON")
	}
	var plain any
	if err := json.Unmarshal(raw, &plain); err != nil {
		return nil, fmt.Errorf("must be valid JSON")
	}
	return plain, nil
}
