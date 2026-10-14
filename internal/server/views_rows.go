package server

import (
	"fmt"
	"html/template"
	"sort"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A listing is rows: a checkbox at the front when the record has a
// yes-or-no field, the title as the link, its facts at the right. Dated
// records come grouped by when, the way a person keeps a list: what is
// overdue, today, this week, later, undated, and what is done. Only what
// can be done is overdue, as its glance says (records/glance_day.go): a
// meeting, a logged entry, a dated note that has passed is Earlier, the
// latest first, and a log of habit entries no longer read as 29 overdue.

// groupOrder is the order the groups come in; empty ones are left out.
var groupOrder = []string{"Overdue", "Today", "This week", "Later", "Earlier", "No date", "Done"}

func (s *Server) rows(t *schema.Type, recs []*store.Record, now time.Time) string {
	dated := t.DayField()
	var b strings.Builder
	// Told apart across the whole listing, every group of it: an agent or
	// a person moving by controls meets them all on one page.
	told := s.recordsApart(t, recs)
	// What is in each, counted for the whole listing at once (records/glance_count.go).
	in := records.CountsOf(s.app.Store, t, recs, now)
	if dated == "" {
		fmt.Fprintf(&b, `<ol class="sw-plain sw-rows" data-dot="%d" aria-label="%s">`, s.dotOf(t.Name), template.HTMLEscapeString(schema.Plural(t.Name)))
		for _, rec := range recs {
			b.WriteString(s.row(t, rec, 2, told[rec.ID], in))
		}
		b.WriteString("</ol>")
		return b.String()
	}
	groups := map[string][]*store.Record{}
	for _, rec := range recs {
		g := whenGroup(t, rec, dated, now)
		groups[g] = append(groups[g], rec)
	}
	latestFirst(groups["Earlier"], dated) // what has passed, the nearest first
	for _, name := range groupOrder {
		list := groups[name]
		if len(list) == 0 {
			continue
		}
		span, pause := "", ""
		if name == "This week" {
			span = fmt.Sprintf(` <span class="sw-group__range">%s to %s</span>`, now.Format("2 Jan"), now.AddDate(0, 0, 6).Format("2 Jan"))
			pause = "," // heard as This week 1 task, 12 Oct to 18 Oct; with nothing after, no comma
		}
		fmt.Fprintf(&b, `<h2 class="sw-group">%s <span class="sw-group__count">%d<span class="sw-visually-hidden"> %s%s</span></span>%s</h2><ol class="sw-plain sw-rows" data-dot="%d" aria-label="%s, %s">`,
			name, len(list), schema.Noun(len(list), t.Name), pause, span, s.dotOf(t.Name), template.HTMLEscapeString(schema.Plural(t.Name)), strings.ToLower(name))
		for _, rec := range list {
			b.WriteString(s.row(t, rec, 3, told[rec.ID], in))
		}
		b.WriteString("</ol>")
	}
	return b.String()
}

// row is one record: its box, its title, its facts. Only a primary toggle
// (done/completed/complete/finished) gets a checkbox in the row; secondary
// settings like pinned or show are shown as badges instead. told is what
// tells it from another row with its title, read after the title by its
// link and its box, or "". in is what is in the listing's records,
// counted once for all of them.
func (s *Server) row(t *schema.Type, rec *store.Record, level int, told string, in records.Counts) string {
	class, box := "sw-row", ""
	if t.DoneField() != nil {
		if props, ok := s.markOf(t, rec); ok {
			props["quiet"] = true
			props["context"] = withContext(str(props["context"], ""), told)
			box = string(s.component("mark", props))
			if on, _ := props["checked"].(bool); on {
				class += " sw-row--done"
			}
		}
	}
	// A title cut short for the row keeps its whole on the link, for a
	// pointer that rests on it.
	full := s.title(t, rec)
	whole := ""
	if short := trim.Title(full); short != full {
		whole = ` title="` + template.HTMLEscapeString(full) + `"`
	}
	// The words that tell it apart stay inside the link, hidden: they are
	// its name, which two rows of one title must not share (WCAG 2.4.9,
	// design/foundations/names.md); the row's facts say them to the eye.
	apartHTML := ""
	if told != "" {
		apartHTML = `<span class="sw-visually-hidden"> (` + template.HTMLEscapeString(told) + `)</span>`
	}
	return fmt.Sprintf(`<li class="%s">%s<h%d class="sw-row__title"><a class="sw-row__link" href="/t/%s/%s"%s>%s%s</a></h%d><p class="sw-row__meta">%s</p></li>`,
		class, box, level, t.Name, rec.ID, whole, template.HTMLEscapeString(trim.Title(full)), apartHTML, level, s.facts(t, rec, factOpts{Boxed: box != "", Counts: in}))
}

// whenGroup says where a record sits in time: done first, because a done
// thing is not overdue whatever its day was.
func whenGroup(t *schema.Type, rec *store.Record, dated string, now time.Time) string {
	if f := t.DoneField(); f != nil {
		if on, _ := rec.Fields[f.Name].(bool); on {
			return "Done"
		}
	}
	v, _ := rec.Fields[dated].(string)
	day, _, ok := when.Stored(v, now) // a whole day its date, a moment in the person's zone
	if v == "" || !ok {
		return "No date"
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	d := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, now.Location())
	switch {
	case d.Before(today) && t.DoneField() == nil:
		return "Earlier" // only what can be done is late
	case d.Before(today):
		return "Overdue"
	case d.Equal(today):
		return "Today"
	case d.Before(today.AddDate(0, 0, 7)):
		return "This week"
	}
	return "Later"
}

// latestFirst orders records by their day, the latest first.
func latestFirst(recs []*store.Record, dated string) {
	sort.SliceStable(recs, func(i, j int) bool {
		a, _ := recs[i].Fields[dated].(string)
		b, _ := recs[j].Fields[dated].(string)
		return a > b
	})
}
