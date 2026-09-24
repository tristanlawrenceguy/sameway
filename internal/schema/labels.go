package schema

import "strings"

// ValueLabel is how a person sees one of an enum field's values: the
// schema's label for it, else the value made readable (in_progress is
// "In progress"). Every place a value is shown or offered uses this, so a
// dropdown and the page it saves to say the same thing.
func (f Field) ValueLabel(value string) string {
	if l := f.Labels[value]; l != "" {
		return l
	}
	s := strings.ReplaceAll(value, "_", " ")
	if s == "" || strings.ToUpper(s) == s {
		return s // empty, or an acronym such as GET
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
