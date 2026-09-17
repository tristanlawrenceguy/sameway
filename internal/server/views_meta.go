package server

import (
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// The few words a page says about a record beside its title: its state,
// the day that matters to it, when it was made.

// whenMade says when a record was made and last changed, as a person reads
// a time, in one quiet line under its fields.
func whenMade(rec *store.Record) string {
	made := when.Text(rec.CreatedAt.UTC().Format(time.RFC3339))
	changed := when.Text(rec.UpdatedAt.UTC().Format(time.RFC3339))
	if changed == made {
		return `<p class="sw-detail__when sw-muted sw-small">Created ` + made + `</p>`
	}
	return `<p class="sw-detail__when sw-muted sw-small">Created ` + made + ` · Updated ` + changed + `</p>`
}

// listMeta is the line under a title in a listing: the record's state when
// it has one, then the day that matters to it, or failing that when it last
// changed. A person scanning a list wants those, not an id or a timestamp.
func listMeta(t *schema.Type, rec *store.Record) string {
	var parts []string
	// Its yes-or-no, when it is yes: done, pinned, whatever the type calls it.
	for _, f := range t.Fields {
		if f.Type == "bool" {
			if v, _ := rec.Fields[f.Name].(bool); v {
				parts = append(parts, capitalize(label(f.Name)))
			}
			break
		}
	}
	for _, f := range t.Fields {
		if f.Type == "enum" {
			if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
				parts = append(parts, capitalize(v))
			}
			break
		}
	}
	dated := false
	for _, f := range t.Fields {
		if f.Type == "datetime" {
			if v, ok := rec.Fields[f.Name].(string); ok && v != "" {
				parts, dated = append(parts, label(f.Name)+" "+when.Text(v)), true
				break
			}
		}
	}
	if !dated {
		parts = append(parts, "Updated "+when.Text(rec.UpdatedAt.UTC().Format(time.RFC3339)))
	}
	return strings.Join(parts, " · ")
}
