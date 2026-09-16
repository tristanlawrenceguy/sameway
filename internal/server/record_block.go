package server

import (
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
// gone rather than rendered as nothing.
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
		out["missing"] = true
		return out, ""
	}
	out["title"] = titleOf(t, rec)
	out["titleProp"] = t.Title
	var fields []any
	for _, f := range t.Fields {
		if f.Name == t.Title {
			continue
		}
		val := display(f, rec.Fields[f.Name])
		if val == "" {
			continue
		}
		if _, have := out["text"]; !have && isText(f) {
			out["text"], out["textProp"] = val, f.Name
			continue
		}
		fields = append(fields, map[string]any{"label": label(f.Name), "value": val})
	}
	if len(fields) > 0 {
		out["fields"] = fields
	}
	return out, "/t/" + t.Name + "/" + rec.ID + "/props"
}

func isText(f schema.Field) bool {
	return f.Type == "text" || f.Type == "markdown"
}
