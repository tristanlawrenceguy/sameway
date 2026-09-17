package server

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// A listing is rows: a checkbox at the front when the record has a
// yes-or-no field, the title as the link, its facts at the right. Dated
// records come grouped by when, the way a person keeps a list: what is
// overdue, today, this week, later, undated, and what is done.

// groupOrder is the order the groups come in; empty ones are left out.
var groupOrder = []string{"Overdue", "Today", "This week", "Later", "No date", "Done"}

func (s *Server) rows(t *schema.Type, recs []*store.Record, now time.Time) string {
	dated := ""
	for _, f := range t.Fields {
		if f.Type == "datetime" {
			dated = f.Name
			break
		}
	}
	var b strings.Builder
	if dated == "" {
		fmt.Fprintf(&b, `<ol class="sw-plain sw-rows" aria-label="%s">`, template.HTMLEscapeString(plural(t.Name)))
		for _, rec := range recs {
			b.WriteString(s.row(t, rec, 2))
		}
		b.WriteString("</ol>")
		return b.String()
	}
	groups := map[string][]*store.Record{}
	for _, rec := range recs {
		g := whenGroup(t, rec, dated, now)
		groups[g] = append(groups[g], rec)
	}
	for _, name := range groupOrder {
		list := groups[name]
		if len(list) == 0 {
			continue
		}
		span := ""
		if name == "This week" {
			span = fmt.Sprintf(`<span class="sw-group__range">%s – %s</span>`, now.Format("2 Jan"), now.AddDate(0, 0, 6).Format("2 Jan"))
		}
		fmt.Fprintf(&b, `<h2 class="sw-group">%s <span class="sw-group__count">%d</span>%s</h2><ol class="sw-plain sw-rows" aria-label="%s, %s">`,
			name, len(list), span, template.HTMLEscapeString(plural(t.Name)), strings.ToLower(name))
		for _, rec := range list {
			b.WriteString(s.row(t, rec, 3))
		}
		b.WriteString("</ol>")
	}
	return b.String()
}

// row is one record: its box, its title, its facts.
func (s *Server) row(t *schema.Type, rec *store.Record, level int) string {
	class, box := "sw-row", ""
	if props, ok := markOf(t, rec); ok {
		props["quiet"] = true
		box = string(s.component("mark", props))
		if on, _ := props["checked"].(bool); on {
			class += " sw-row--done"
		}
	}
	return fmt.Sprintf(`<li class="%s">%s<h%d class="sw-row__title"><a class="sw-row__link" href="/t/%s/%s">%s</a></h%d><p class="sw-row__meta">%s</p></li>`,
		class, box, level, t.Name, rec.ID, template.HTMLEscapeString(titleOf(t, rec)), level, s.facts(t, rec, factOpts{Boxed: box != ""}))
}

// whenGroup says where a record sits in time: done first, because a done
// thing is not overdue whatever its day was.
func whenGroup(t *schema.Type, rec *store.Record, dated string, now time.Time) string {
	for _, f := range t.Fields {
		if f.Type == "bool" {
			if on, _ := rec.Fields[f.Name].(bool); on {
				return "Done"
			}
			break
		}
	}
	v, _ := rec.Fields[dated].(string)
	ts, err := time.Parse(time.RFC3339, v)
	if v == "" || err != nil {
		return "No date"
	}
	day := ts.Local()
	if strings.HasSuffix(v, "T00:00:00Z") {
		day = ts.UTC()
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, now.Location())
	switch {
	case d.Before(today):
		return "Overdue"
	case d.Equal(today):
		return "Today"
	case d.Before(today.AddDate(0, 0, 7)):
		return "This week"
	}
	return "Later"
}
