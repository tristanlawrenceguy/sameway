package schema

import "strings"

// irregular plurals English does not form by rule.
var irregular = map[string]string{"person": "people", "child": "children", "mouse": "mice", "man": "men", "woman": "women"}

// Words is a content type's name as a person reads it: a name is stored
// with underscores (test_type) and said with spaces (test type). Every
// place a type is named to people, in a button, a heading, a crumb or a
// log line, says it through this or through Plural, which uses it.
func Words(name string) string {
	return strings.ReplaceAll(name, "_", " ")
}

// Plural is the plural of a content type's name, for headings, navigation
// and sentences: activity, activities; box, boxes; person, people;
// test_type, test types. A name that already ends in s is taken to be
// plural already (news, series).
func Plural(name string) string {
	name = Words(name)
	if p, ok := irregular[name]; ok {
		return p
	}
	switch {
	case name == "" || strings.HasSuffix(name, "s"):
		return name
	case strings.HasSuffix(name, "x"), strings.HasSuffix(name, "z"), strings.HasSuffix(name, "ch"), strings.HasSuffix(name, "sh"):
		return name + "es"
	case len(name) >= 2 && strings.HasSuffix(name, "y") && !strings.ContainsRune("aeiou", rune(name[len(name)-2])):
		return name[:len(name)-1] + "ies"
	}
	return name + "s"
}
