package jsontext

import (
	"encoding/json"
	"testing"
)

// <, > and & are written as they are, and what is written reads back as
// json does; indented when asked.
func TestJSONAsText(t *testing.T) {
	t.Parallel()
	v := map[string]any{"where": []string{"due<today", "amount>=3"}, "title": "Tom & Jerry", "key": "Bearer <key>"}
	raw, err := Marshal(v)
	if err != nil || string(raw) != `{"key":"Bearer <key>","title":"Tom & Jerry","where":["due<today","amount>=3"]}` {
		t.Errorf("as written: %s %v", raw, err)
	}
	var back map[string]any
	if err := json.Unmarshal(raw, &back); err != nil || back["title"] != "Tom & Jerry" {
		t.Errorf("reads back: %v %v", back, err)
	}
	if ind, _ := MarshalIndent(map[string]any{"a": "<"}, "", "  "); string(ind) != "{\n  \"a\": \"<\"\n}" {
		t.Errorf("indented: %q", ind)
	}
}
