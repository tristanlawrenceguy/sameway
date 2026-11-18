package server

import (
	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// says is what a record says of itself under its title, the same on its
// own page and as a block on the canvas: its text, the first structured
// field with words in it, and its other fields with something in them,
// less what the heading and the chips already say, what nobody chose (a
// default; schema.Unchosen) and what means nothing to a person (a file's
// storage path, a habit's goal of 0). The canvas
// once chose for itself and showed both.
func (s *Server) says(t *schema.Type, rec *store.Record) (text string, fields []schema.Field) {
	for _, f := range t.Shown() {
		if f.Type == "markdown" {
			if s.display(f, rec.Fields[f.Name]) != "" {
				text = f.Name
			}
			break
		}
	}
	head := s.headFields(t, rec) // glance.go
	for _, f := range t.Shown() {
		if f.Name == text || head[f.Name] || s.display(f, rec.Fields[f.Name]) == "" || f.Unchosen(rec.Fields[f.Name]) || noGoal(t.Name, f.Name, rec.Fields[f.Name]) || (f.Name == "currency" && hasSum(t)) {
			continue
		}
		fields = append(fields, f)
	}
	return text, fields
}

// fieldSays is a field's value as its page lists it. A sum is said with
// its currency, as one: Amount 60.00 GBP. The page once said Amount 60
// and left the currency out as the default, so a person heard a number
// and not what it was in.
func (s *Server) fieldSays(f schema.Field, rec *store.Record) string {
	n, isSum := rec.Fields[f.Name].(float64)
	if cur, _ := rec.Fields["currency"].(string); isSum && f.Type == "float" && cur != "" {
		return chat.Money(n, cur)
	}
	return s.display(f, rec.Fields[f.Name])
}

// hasSum says whether a type keeps an amount beside its currency, which
// then says the currency with it.
func hasSum(t *schema.Type) bool {
	for _, f := range t.Shown() {
		if f.Type == "float" {
			return true
		}
	}
	return false
}
