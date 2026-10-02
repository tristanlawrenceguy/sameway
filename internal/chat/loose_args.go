package chat

import (
	"encoding/json"
	"strings"
)

// A smaller model reaches for the shapes it knows: a record's fields
// beside type instead of inside fields, as a JSON string, or under set.
// What it meant is plain, so it is taken as meant, where an error would
// cost it a turn or five.

// loosen rewrites a call's arguments into the shape its tool takes, or
// returns them as they were.
func loosen(tool string, raw json.RawMessage) json.RawMessage {
	var m map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &m) != nil || m == nil {
		return raw
	}
	changed := false
	switch tool {
	case "create_record", "update_record":
		if s, ok := m["fields"].(string); ok {
			if obj := jsonObject(s); obj != nil {
				m["fields"], changed = obj, true
			}
		}
		if _, ok := m["fields"]; !ok {
			fields := map[string]any{}
			for k, v := range m {
				if k == "type" || k == "id" || k == "version" {
					continue
				}
				delete(m, k)
				if s, ok := v.(string); ok {
					if obj := jsonObject(s); obj != nil {
						for fk, fv := range obj {
							fields[fk] = fv
						}
						continue
					}
				}
				if obj, ok := v.(map[string]any); ok && (k == "set" || k == "values" || k == "record" || k == "data") {
					for fk, fv := range obj {
						fields[fk] = fv
					}
					continue
				}
				fields[k] = v
			}
			if len(fields) > 0 {
				m["fields"], changed = fields, true
			}
		}
	}
	if !changed {
		return raw
	}
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}

// jsonObject is s read as a JSON object, or nil.
func jsonObject(s string) map[string]any {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") {
		return nil
	}
	var obj map[string]any
	if json.Unmarshal([]byte(s), &obj) != nil {
		return nil
	}
	return obj
}
