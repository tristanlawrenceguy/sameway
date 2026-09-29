package server

import (
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// says is what a record says of itself under its title, the same on its
// own page and as a block on the canvas: its text, the first structured
// field with words in it, and its other fields with something in them,
// less what the heading and the chips already say and what means nothing
// to a person (a file's storage path, a habit's goal of 0). The canvas
// once chose for itself and showed both.
func (s *Server) says(t *schema.Type, rec *store.Record) (text string, fields []schema.Field) {
	for _, f := range t.Shown() {
		if f.Type == "markdown" {
			if display(f, rec.Fields[f.Name]) != "" {
				text = f.Name
			}
			break
		}
	}
	head := headFields(t, rec)
	for _, f := range t.Shown() {
		if f.Name == text || head[f.Name] || display(f, rec.Fields[f.Name]) == "" || noGoal(t.Name, f.Name, rec.Fields[f.Name]) {
			continue
		}
		fields = append(fields, f)
	}
	return text, fields
}
