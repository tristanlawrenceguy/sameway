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
