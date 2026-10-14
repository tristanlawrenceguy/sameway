package schema

import "strings"

// Several names in one line. An entry is called by its habit and how much
// ("Running miles, 5 miles"), a person can be "Silva, Ana", and three
// such names joined by commas read as six (crew finding 0748). Every list
// of names a person reads goes through Names or AndNames, so the next
// place that joins names cannot forget; tools/check fails a join of names
// or titles by a bare comma.

// Between is what goes between names: a comma, or a semicolon when any
// name has a comma of its own.
func Between(names []string) string {
	for _, n := range names {
		if strings.Contains(n, ",") {
			return "; "
		}
	}
	return ", "
}

// Names is names in a row, set apart as Between says.
func Names(names []string) string {
	return strings.Join(names, Between(names))
}

// AndNames is names as said: "a, b and c", or "a, 1; b, 2; and c, 3" when
// a name has a comma, so the last is as plainly apart as the rest.
func AndNames(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	sep, last := Between(names), " and "
	if sep == "; " {
		last = "; and "
	}
	return strings.Join(names[:len(names)-1], sep) + last + names[len(names)-1]
}
