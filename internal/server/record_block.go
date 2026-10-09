package server

import (
	"html/template"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
)

// recordComponent is the block that shows a content record on the canvas.
// A block of it holds only the type and the id; the words are the record's
// own, read here when the page renders, so the note on the canvas and the
// note on /t/note are one thing and never a copy that drifts.
const recordComponent = "record"

// resolveRecord fills a record block's props from the store: the title, the
// main text, and the other fields with something in them. It also says
// where the inline editor should post, which is the record's own props
// endpoint rather than the block's. A record that is gone is said to be
// gone rather than rendered as nothing. The second answer is what the
// editor needs, the same as on the record's own page (editing).
func (s *Server) resolveRecord(props map[string]any) (map[string]any, string) {
	out := map[string]any{}
	for k, v := range props {
		out[k] = v
	}
	typeName, _ := props["type"].(string)
	id, _ := props["record"].(string)
	t, ok := s.app.Types.Get(typeName)
	if !ok {
		out["missing"] = true
		return out, ""
	}
	rec, err := s.app.Store.Get(t.Name, id)
	if err != nil {
		// Gone, with the way on: the rest of its list.
		out["missing"] = true
		out["listHref"], out["listLabel"] = "/t/"+t.Name, "See all "+schema.Plural(t.Name)
		return out, ""
	}
	out["kind"] = capitalize(schema.Words(t.Name))
	out["title"] = s.title(t, rec)
	out["titleProp"] = t.Title
	// What it says is what its page says (record_says.go): the same text,
	// the same fields, each said as the page says it.
	text, shown := s.says(t, rec)
	if f, ok := t.Field(text); ok {
		// Structured text shows its structure here as it does on the
		// record's page, and is edited the same way.
		out["text"], out["textProp"], out["structured"] = s.display(*f, rec.Fields[f.Name]), f.Name, true
	}
	var fields []any
	for _, f := range shown {
		val := s.display(f, rec.Fields[f.Name])
		item := s.fieldItem(t, f, rec.Fields[f.Name], val)
		if v, ok := item["value"].(string); ok && v != "" {
			val = v // a ref or what a reminder is about, by its title
		}
		fields = append(fields, map[string]any{"label": f.Display(), "value": val})
	}
	if len(fields) > 0 {
		out["fields"] = fields
	}
	if actions := s.markActions(t, rec); actions != nil {
		out["actions"] = actions
	}
	return out, editing(t, rec)
}

// recordEditFields is every field of a record block's record, for its
// editor: on the canvas, Edit offers what the record's own page does, the
// facts as well as the title and the text, the empty ones behind Add.
func (s *Server) recordEditFields(name string, props map[string]any) template.HTML {
	if name != recordComponent || props["missing"] == true {
		return ""
	}
	typeName, _ := props["type"].(string)
	id, _ := props["record"].(string)
	t, ok := s.app.Types.Get(typeName)
	if !ok {
		return ""
	}
	rec, err := s.app.Store.Get(t.Name, id)
	if err != nil {
		return ""
	}
	return template.HTML(s.editFields(t, rec))
}
