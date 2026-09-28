package chat

// obj is a JSON schema for an object with these properties. Strict
// servers (llama.cpp builds a grammar from this) reject "required": null,
// so the key is only present when there is a list.
func obj(props map[string]any, required ...string) map[string]any {
	s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		s["required"] = required
	}
	return s
}
