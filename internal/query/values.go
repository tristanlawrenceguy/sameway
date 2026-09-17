package query

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The record's own times are fields too, for ordering and for "changed
// since".
var own = map[string]string{"created_at": "datetime", "updated_at": "datetime"}

func known(t *schema.Type, field string) bool {
	if _, ok := own[field]; ok {
		return true
	}
	_, ok := t.Field(field)
	return ok
}

func fieldNames(t *schema.Type) []string {
	var out []string
	for _, f := range t.Fields {
		out = append(out, f.Name)
	}
	return append(out, "created_at", "updated_at")
}

func kindOf(t *schema.Type, field string) string {
	if k, ok := own[field]; ok {
		return k
	}
	if f, ok := t.Field(field); ok {
		return f.Type
	}
	return "string"
}

func valueOf(rec *store.Record, field string) any {
	switch field {
	case "created_at":
		return rec.CreatedAt.UTC().Format(time.RFC3339)
	case "updated_at":
		return rec.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return rec.Fields[field]
}

func empty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(x) == ""
	case bool:
		return !x
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	n, ok := number(v)
	return ok && n == 0
}

func number(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return n, err == nil
	}
	return 0, false
}

func text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []any, map[string]any:
		raw, _ := json.Marshal(x)
		return string(raw)
	}
	return fmt.Sprint(v)
}

// parseDate reads a date the way a person writes one: a day, a moment, or
// a distance from now. day says whether it names a whole day rather than
// an instant, so "due=tomorrow" means any time tomorrow.
func parseDate(s string, now time.Time) (t time.Time, day bool, ok bool) {
	return when.Parse(s, now)
}
