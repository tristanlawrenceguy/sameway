package ui

import (
	"html/template"
	"strings"
)

// FromField is the field naming the page an action returns to afterwards,
// and BackField the place on that page it was taken (internal/server's
// back.go), so the person comes back to where they were.
const (
	FromField = "from"
	BackField = "back"
)

// Form is a form that does one action, its hidden fields and its button,
// written the same way everywhere: posted unless it says get, every value
// escaped, the page and place to come back to under the names the server
// reads, and its button a submit button.
type Form struct {
	Action  string
	Get     bool   // a form that only asks, such as a search; posted otherwise
	Class   string // the form's own class, such as sw-stack
	Enctype string // multipart/form-data for a file
	Target  string // _blank, opened with rel="noopener"
	From    string // the page to return to; the action's own when empty
	Back    string // the id of the place on that page the form sits in
	Hidden  []Field
	// Body is what comes before the button: the fields a person fills.
	Body template.HTML
	// Button sends it. A form with no Button ends with its Body.
	Button *Button
	// After is what follows the button inside the form, such as a line
	// saying what pressing it does.
	After template.HTML
}

// Field is one hidden name and value.
type Field struct{ Name, Value string }

// Hidden is a list of hidden fields from names and values in turn.
func Hidden(nameValue ...string) []Field {
	var out []Field
	for i := 0; i+1 < len(nameValue); i += 2 {
		out = append(out, Field{nameValue[i], nameValue[i+1]})
	}
	return out
}

// Open is the form's start tag and its hidden fields.
func (f Form) Open() string {
	esc := template.HTMLEscapeString
	var b strings.Builder
	method := "post"
	if f.Get {
		method = "get"
	}
	b.WriteString(`<form method="` + method + `" action="` + esc(f.Action) + `"`)
	if f.Enctype != "" {
		b.WriteString(` enctype="` + esc(f.Enctype) + `"`)
	}
	if f.Target != "" {
		b.WriteString(` target="` + esc(f.Target) + `" rel="noopener"`)
	}
	if f.Class != "" {
		b.WriteString(` class="` + esc(f.Class) + `"`)
	}
	b.WriteString(`>`)
	fields := f.Hidden
	if f.Back != "" {
		fields = append([]Field{{BackField, f.Back}}, fields...)
	}
	if f.From != "" {
		fields = append([]Field{{FromField, f.From}}, fields...)
	}
	for _, h := range fields {
		b.WriteString(`<input type="hidden" name="` + esc(h.Name) + `" value="` + esc(h.Value) + `">`)
	}
	return b.String()
}

// HTML is the whole form, its button rendered by render.
func (f Form) HTML(render Renderer) template.HTML {
	out := f.Open() + string(f.Body)
	if f.Button != nil {
		btn := *f.Button
		if btn.Type == "" {
			btn.Type = Submit
		}
		out += string(render(btn))
	}
	return template.HTML(out + string(f.After) + `</form>`)
}
