package server

import (
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A record's page lists what is in it: the records of each ref the schema
// marks listed that point at it (withins, glance_count.go), the same rule
// its glance counts by. A project lists its tasks, a person its
// interactions, a meeting the tasks that came up at it. It was a section
// written for projects by name; any type can now have one, by its schema,
// and older workspaces get the starter's flags (schema Complete). The same
// records are one of the record's connections (related.go, points-here);
// a connection listed here is not opened there again.

// backrefs is each listed back-reference to rec as a list under its own
// heading, newest change first, saying so when it is empty, and the
// connection keys it shows.
func (s *Server) backrefs(t *schema.Type, rec *store.Record) (string, []string) {
	var b strings.Builder
	var keys []string
	for _, w := range s.withins(t) {
		u, f := w.t, w.field
		id := strings.ReplaceAll(schema.Plural(u.Name), " ", "-") + "-" + rec.ID
		if f != t.Name {
			id += "-" + slugKey(f)
		}
		props := s.resolveCollection(map[string]any{
			"type":  u.Name,
			"where": []string{f + "=" + rec.ID},
			"order": "-updated_at",
			"label": capitalize(schema.Plural(u.Name)),
			"id":    id,
		}, "")
		delete(props, "summary") // the template says "Nothing here yet." when empty
		b.WriteString(string(s.component(collectionComponent, props)))
		keys = append(keys, relate.PointsHere+":"+u.Name+"."+f)
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
