package server

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The few words a page says about a record beside its title: its state,
// the day that matters to it, when it was made.

// lede is the line under a record's title: its facts as chips, then when
// it was made.
func (s *Server) lede(t *schema.Type, rec *store.Record) template.HTML {
	return template.HTML(`<p class="sw-lede">` + s.facts(t, rec, true) + `</p>`)
}

// howMany says how many there are, under a listing's title.
func howMany(n int, typeName string) template.HTML {
	what := plural(typeName)
	if n == 1 {
		what = typeName
	}
	return template.HTML(fmt.Sprintf(`<p class="sw-lede">%d %s</p>`, n, template.HTMLEscapeString(what)))
}

// facts is what a person wants to know about a record at a glance, as
// chips whose colour says the same thing the words do: done in green,
// a state in blue, a day that has passed in amber. When there is
// nothing of the kind, when it last changed. made adds when it was made.
func (s *Server) facts(t *schema.Type, rec *store.Record, made bool) string {
	var parts []string
	done := false
	for _, f := range t.Fields {
		if f.Type == "bool" {
			if v, _ := rec.Fields[f.Name].(bool); v {
				done = true
				parts = append(parts, string(s.component("badge", map[string]any{"label": capitalize(label(f.Name)), "tone": "success"})))
			}
			break
		}
	}
	for _, f := range t.Fields {
		if f.Type == "enum" {
			if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
				parts = append(parts, string(s.component("badge", map[string]any{"label": capitalize(v), "tone": "info"})))
			}
			break
		}
	}
	dated := false
	for _, f := range t.Fields {
		if f.Type == "datetime" {
			if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
				dated = true
				ts, _ := time.Parse(time.RFC3339, v)
				text, tone := label(f.Name)+" "+when.Text(v), "neutral"
				if !done && ts.Before(time.Now()) && !strings.HasSuffix(v, "T00:00:00Z") || !done && strings.HasSuffix(v, "T00:00:00Z") && ts.AddDate(0, 0, 1).Before(time.Now()) {
					text, tone = "Was "+strings.ToLower(label(f.Name))+" "+when.Text(v), "warning"
				}
				parts = append(parts, string(s.component("badge", map[string]any{"label": text, "tone": tone})))
				break
			}
		}
	}
	if !dated && !made {
		parts = append(parts, `<span class="sw-muted">Updated `+when.Text(rec.UpdatedAt.UTC().Format(time.RFC3339))+`</span>`)
	}
	if made {
		parts = append(parts, whenMade(rec))
	}
	return strings.Join(parts, " ")
}

// whenMade says when a record was made and last changed, as a person reads
// a time, in one quiet line under its fields.
func whenMade(rec *store.Record) string {
	made := when.Text(rec.CreatedAt.UTC().Format(time.RFC3339))
	changed := when.Text(rec.UpdatedAt.UTC().Format(time.RFC3339))
	if changed == made {
		return `<span class="sw-detail__when sw-muted sw-small">Created ` + made + `</span>`
	}
	return `<span class="sw-detail__when sw-muted sw-small">Created ` + made + ` · Updated ` + changed + `</span>`
}
