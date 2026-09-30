package chat

import (
	"encoding/json"
	"strings"
)

// ForModel is a props or fields schema as a model writing it needs it: the
// properties the server fills in itself (a collection's items, a calendar's
// nav, a record's title) are left out, since a model that reads them either
// ignores them or, worse, writes them. Everything the model may give is
// kept as it was. Validation still takes the whole schema; this only cuts
// what the prompt and describe repeat.
func ForModel(raw json.RawMessage) json.RawMessage {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	out, err := json.Marshal(trimFilled(v))
	if err != nil {
		return raw
	}
	return out
}

// trimFilled walks a schema and drops the server-filled properties.
func trimFilled(v any) any {
	switch o := v.(type) {
	case map[string]any:
		if props, ok := o["properties"].(map[string]any); ok {
			dropped := map[string]bool{}
			for name, p := range props {
				if d, ok := p.(map[string]any); ok && ServerFilled(str(d["description"])) {
					delete(props, name)
					dropped[name] = true
				}
			}
			if req, ok := o["required"].([]any); ok && len(dropped) > 0 {
				kept := []any{}
				for _, r := range req {
					if s, _ := r.(string); !dropped[s] {
						kept = append(kept, r)
					}
				}
				o["required"] = kept
			}
		}
		for k, x := range o {
			o[k] = trimFilled(x)
		}
	case []any:
		for i, x := range o {
			o[i] = trimFilled(x)
		}
	}
	return v
}

// ServerFilled says whether a property's description says the server, not
// the writer, gives it: "Filled in by the server: ...", "... Leave out.",
// "... kept by the system." A property the server fills only in some case
// ("when type is given") stays, since the writer gives it otherwise.
func ServerFilled(desc string) bool {
	d := strings.TrimSpace(desc)
	switch {
	case d == "":
		return false
	case strings.HasPrefix(d, "Filled in by the server"):
		return true
	case strings.Contains(d, "Filled in by the server") && strings.HasSuffix(d, "Leave out."):
		return true
	case strings.HasSuffix(d, "Filled in by the server.") || strings.HasSuffix(d, "Filled in by the server from the records."):
		return true
	case strings.Contains(d, "kept by the system"):
		return true
	}
	return false
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
