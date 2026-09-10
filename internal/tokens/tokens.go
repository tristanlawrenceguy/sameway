// Package tokens turns design/tokens/tokens.json into CSS custom properties.
//
// The JSON file is the single source of truth. tokens.css is generated from it
// and checked in, and `go run ./tools/check` fails when the two drift apart.
package tokens

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Prefix is the CSS custom property prefix shared by every token.
const Prefix = "--sw-"

// Generate renders the CSS for a tokens.json document.
//
// Scalar tokens become `--sw-<group>-<name>` on :root. Color tokens with
// "light" and "dark" values get a light default plus two dark overrides: one
// for prefers-color-scheme and one for an explicit data-theme="dark".
func Generate(src []byte) (string, error) {
	var doc map[string]any
	if err := json.Unmarshal(src, &doc); err != nil {
		return "", fmt.Errorf("tokens.json: %w", err)
	}
	var light, dark []string
	for _, group := range sortedKeys(doc) {
		if strings.HasPrefix(group, "$") {
			continue
		}
		entries, ok := doc[group].(map[string]any)
		if !ok {
			return "", fmt.Errorf("tokens.json: group %q must be an object", group)
		}
		for _, name := range sortedKeys(entries) {
			prop := Prefix + group + "-" + name
			switch v := entries[name].(type) {
			case string:
				light = append(light, fmt.Sprintf("  %s: %s;", prop, v))
			case map[string]any:
				l, d := v["light"], v["dark"]
				if l == nil || d == nil {
					return "", fmt.Errorf("tokens.json: %s.%s needs light and dark", group, name)
				}
				light = append(light, fmt.Sprintf("  %s: %v;", prop, l))
				dark = append(dark, fmt.Sprintf("  %s: %v;", prop, d))
			default:
				return "", fmt.Errorf("tokens.json: %s.%s has unsupported value", group, name)
			}
		}
	}
	var b strings.Builder
	b.WriteString("/* Generated from design/tokens/tokens.json by `go run ./tools/tokens`. Do not edit. */\n")
	b.WriteString(":root {\n  color-scheme: light dark;\n")
	b.WriteString(strings.Join(light, "\n"))
	b.WriteString("\n}\n\n@media (prefers-color-scheme: dark) {\n  :root:not([data-theme=\"light\"]) {\n")
	b.WriteString(indent(strings.Join(dark, "\n")))
	b.WriteString("\n  }\n}\n\n:root[data-theme=\"dark\"] {\n")
	b.WriteString(strings.Join(dark, "\n"))
	b.WriteString("\n}\n")
	return b.String(), nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func indent(s string) string {
	return "  " + strings.ReplaceAll(s, "\n", "\n  ")
}
