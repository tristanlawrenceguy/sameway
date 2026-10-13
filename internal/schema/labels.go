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

// Display is what a person calls a field: the schema's label, else its name
// made readable (follow_up is "Follow up"). A form, a column, a heading, an
// export's header and an answer to an agent all name a field through this.
func (f Field) Display() string {
	if f.Label != "" {
		return f.Label
	}
	s := Words(f.Name)
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// Words is Display inside a sentence: "3 by follow up", "no task has a
// goal by". An all-capitals label, an acronym such as URL, stays so.
func (f Field) Words() string {
	d := f.Display()
	if strings.ToUpper(d) == d {
		return d
	}
	return strings.ToLower(d)
}

// FieldDisplay is Display for a field named on a type, and the name made
// readable for one the type does not have.
func (t *Type) FieldDisplay(name string) string {
	if f, ok := t.Field(name); ok {
		return f.Display()
	}
	return Field{Name: name}.Display()
}

// FieldWords is Words for a field named on a type, as FieldDisplay.
func (t *Type) FieldWords(name string) string {
	if f, ok := t.Field(name); ok {
		return f.Words()
	}
	return Field{Name: name}.Words()
}

// Kind is a field's kind in a word or two, as a model is told it.
func (f Field) Kind() string {
	req := ""
	if f.Required {
		req = ", required"
	}
	switch {
	case f.RefList():
		return "ids of " + f.To + req
	case f.Type == "ref":
		return "id of " + f.To + req
	case f.Type == "enum":
		return strings.Join(f.Values, "|") + req
	case f.Type == "datetime":
		return "date" + req
	case f.Type == "markdown", f.Type == "text", f.Type == "string":
		return "text" + req
	case f.Type == "bool":
		return "yes/no" + req
	case f.Type == "int", f.Type == "float":
		return "number" + req
	}
	return f.Type + req
}

// ServerFilled says whether a property's description says the server, not
// the writer, gives it: "Filled in by the server: ...", "... Leave out.",
// "... kept by the system." A property the server fills only in some case
// ("when type is given") stays, since the writer gives it otherwise.
func ServerFilled(desc string) bool {
	d := strings.TrimSpace(desc)
	switch {
	case d == "":
		return false
	case strings.HasPrefix(d, "Filled in by the server"):
		return true
	case strings.Contains(d, "Filled in by the server") && strings.HasSuffix(d, "Leave out."):
		return true
	case strings.HasSuffix(d, "Filled in by the server.") || strings.HasSuffix(d, "Filled in by the server from the records."):
		return true
	case strings.Contains(d, "kept by the system"):
		return true
	}
	return false
}
