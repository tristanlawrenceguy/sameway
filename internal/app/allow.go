package app

import "strings"

// allowList is the programs a command action may run, as workspace.yaml
// names them: comma or space separated, blanks dropped.
func allowList(v string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
