package workspace

import (
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// How a setting is written into workspace.yaml: one line changed in its
// place, so the comments a person reads there survive.

// yamlScalar writes a string the way YAML needs it on one line: quoted
// when it has to be, a block when it has lines of its own.
func yamlScalar(value string) string {
	if strings.Contains(value, "\n") {
		return "|-\n" + "    " + strings.ReplaceAll(value, "\n", "\n    ")
	}
	out, err := yaml.Marshal(value)
	if err != nil {
		return strconv.Quote(value)
	}
	return strings.TrimRight(string(out), "\n")
}

// setLine changes one line of a workspace.yaml: the key in its section,
// added at the section's end when it is not there, with the section
// added at the file's end when that is not there either.
func setLine(src, key, scalar string) string {
	section, name := "", key
	if i := strings.Index(key, "."); i > 0 {
		section, name = key[:i], key[i+1:]
	}
	indent := ""
	if section != "" {
		indent = "  "
	}
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	var out []string
	done, in := false, section == ""
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		top := line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "#")
		switch {
		case section != "" && strings.HasPrefix(line, section+":"):
			in = true
		case section != "" && in && top:
			if !done {
				out, done = append(out, indent+name+": "+scalar), true
			}
			in = false
		}
		if in && !done && strings.HasPrefix(trim, name+":") && (section != "" || top) {
			line, done = indent+name+": "+scalar, true
		}
		out = append(out, line)
	}
	if !done {
		if section != "" && !in {
			out = append(out, section+":")
		}
		out = append(out, indent+name+": "+scalar)
	}
	return strings.Join(out, "\n")
}

// topicsOf reads a list of topics as a person gives it, comma separated,
// or as workspace.yaml holds it, [a, b], which is how undo gives it back.
func topicsOf(value string) []string {
	var list []string
	if strings.HasPrefix(value, "[") && yaml.Unmarshal([]byte(value), &list) == nil {
		return list
	}
	for _, t := range strings.Split(value, ",") {
		if t = strings.TrimSpace(t); t != "" {
			list = append(list, t)
		}
	}
	return list
}

// flowList writes a list on one line, each item quoted: ["a", "b"].
func flowList(list []string) string {
	quoted := make([]string, len(list))
	for i, t := range list {
		quoted[i] = strconv.Quote(t)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
