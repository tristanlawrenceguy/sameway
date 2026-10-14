// Package jsontext writes JSON for a reader that reads it as text: a model
// given a tool's result or its prompt, an agent reading an API answer as it
// came. encoding/json writes <, > and & as \u003c, \u003e and \u0026, a
// guard for JSON put inside an HTML page; none of this is, and a model read
// the API's own instructions as "Bearer \u003ckey\u003e" and a condition as
// due\u003ctoday, which pasted back is refused. Here they are as written.
package jsontext

import (
	"bytes"
	"encoding/json"
)

// Marshal is json.Marshal with <, > and & as they are.
func Marshal(v any) ([]byte, error) { return MarshalIndent(v, "", "") }

// MarshalIndent is json.MarshalIndent with <, > and & as they are.
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if prefix != "" || indent != "" {
		enc.SetIndent(prefix, indent)
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}
