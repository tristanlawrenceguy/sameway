package server

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A record's page lists the records whose ref the schema marks listed and
// that point at it: a project's tasks, by task.project's listed. It was a
// section written for projects by name; any type can now have one, by its
// schema, and older workspaces get the starter's flag (schema Complete).
// The same records are one of the record's connections (related.go,
// points-here); a connection listed here is not opened there again.

// backrefs is each listed back-reference to rec as a list under its own
// heading, newest change first, saying so when it is empty, and the
// connection keys it shows.
func (s *Server) backrefs(t *schema.Type, rec *store.Record) (string, []string) {
	var b strings.Builder
	var keys []string
	for _, u := range s.app.Types.Types {
		if u.Internal {
			continue
		}
		for _, f := range u.Shown() {
			if !f.Listed || f.RefTo() != t.Name {
				continue
			}
			id := strings.ReplaceAll(schema.Plural(u.Name), " ", "-") + "-" + rec.ID
			if f.Name != t.Name {
				id += "-" + slugKey(f.Name)
			}
			props := s.resolveCollection(map[string]any{
				"type":  u.Name,
				"where": []string{f.Name + "=" + rec.ID},
				"order": "-updated_at",
				"label": capitalize(schema.Plural(u.Name)),
				"id":    id,
			}, "")
			delete(props, "summary") // the template says "Nothing here yet." when empty
			b.WriteString(string(s.component(collectionComponent, props)))
			keys = append(keys, relate.PointsHere+":"+u.Name+"."+f.Name)
		}
	}
	return b.String(), keys
}

// less is keys without those in drop.
func less(keys, drop []string) []string {
	var out []string
	for _, k := range keys {
		if !has(drop, k) {
			out = append(out, k)
		}
	}
	return out
}
