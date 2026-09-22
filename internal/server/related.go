package server

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/relate"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/when"
)

// A record's connections, at the foot of its page.
//
// internal/relate works out everything this record joins on to. What a
// person sees of it is one line: how many of each, and nothing else. A
// record's page used to open every one of those lists at once, which is a
// page of other people's records with the record you came for at the top
// of it.
//
// Opening one is a link to this same page with ?show=<key>, which renders
// that connection in full where the person already is, so the list is one
// step away rather than a page away, and the address holds what is open:
// the assistant hands over /t/project/x?show=points-here:task.project when
// it has a reason to, and the person can send the same address back.
//
// Nothing is hidden by this. What a sighted person cannot see is not in
// the accessibility tree either, and the one link that opens it is the
// same one link for everyone. This is the other case from the quiet
// layer, which is shown to everybody and only faded.

// related is the connections section: the line of counts, then whichever
// connections the address asked to open.
func (s *Server) related(t *schema.Type, rec *store.Record, show []string) string {
	links := relate.Of(s.app.Store, t, rec, time.Now())
	if len(links) == 0 {
		return ""
	}
	page := "/t/" + t.Name + "/" + rec.ID
	open := opened(links, show)
	var row, lists strings.Builder
	for _, l := range links {
		if has(open, l.Key) {
			lists.WriteString(s.openLink(l, page, open))
			continue
		}
		fmt.Fprintf(&row, `<li><a class="sw-link" href="%s">%s</a></li>`,
			template.HTMLEscapeString(showURL(page, append(append([]string{}, open...), l.Key))),
			template.HTMLEscapeString(s.words(l)))
	}
	var b strings.Builder
	b.WriteString(`<h2 class="sw-visually-hidden">Related</h2>`)
	if row.Len() > 0 {
		b.WriteString(`<nav class="sw-related" aria-label="Related"><ul class="sw-plain sw-related__list">` + row.String() + `</ul></nav>`)
	}
	b.WriteString(lists.String())
	return b.String()
}

// openLink is one connection opened in place: the records it names, as
// the same collection a block would show, with the way to close it again
// in the quiet layer beside its heading.
func (s *Server) openLink(l relate.Link, page string, open []string) string {
	props := s.resolveCollection(map[string]any{
		"type": l.Type, "where": l.Where, "order": l.Order, "limit": 50,
		"label": capitalize(s.words(l)), "level": 2, "id": "related-" + slugKey(l.Key),
	})
	hide := s.component("link", map[string]any{
		"href": showURL(page, without(open, l.Key)), "label": "Hide", "context": s.words(l), "look": "button",
	})
	return fmt.Sprintf(`<div class="sw-related__open" data-related="%s" data-dot="%d">%s<p class="sw-quiet sw-related__hide">%s</p></div>`,
		template.HTMLEscapeString(l.Key), s.dotOf(l.Type), s.component(collectionComponent, props), hide)
}

// words is a connection in a person's words: how many, of what, and what
// makes them related, short enough to read in a line.
func (s *Server) words(l relate.Link) string {
	thing := plural(l.Type)
	if l.Count == 1 {
		thing = l.Type
	}
	count := fmt.Sprintf("%d %s", l.Count, thing)
	switch l.Kind {
	case relate.About:
		return count + " about this"
	case relate.Alongside:
		other := fmt.Sprintf("%d other %s", l.Count, thing)
		if pt, parent, ok := s.aboutOf(l.Through); ok {
			return other + " in " + s.title(pt, parent)
		}
		return other
	case relate.SameDay:
		return count + " on " + when.Text(l.Day+"T00:00:00Z")
	}
	return count
}

// opened is the connections the address asked for, in the order they were
// worked out, ignoring a key for a connection this record does not have.
func opened(links []relate.Link, show []string) []string {
	var out []string
	for _, l := range links {
		if has(show, l.Key) && !has(out, l.Key) {
			out = append(out, l.Key)
		}
	}
	return out
}

// showURL is this page with a set of connections open.
func showURL(page string, keys []string) string {
	if len(keys) == 0 {
		return page
	}
	q := url.Values{}
	for _, k := range keys {
		q.Add("show", k)
	}
	return page + "?" + q.Encode()
}

func has(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

func without(keys []string, key string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != key {
			out = append(out, k)
		}
	}
	return out
}

// slugKey is a connection's key as an element id.
func slugKey(key string) string {
	return strings.NewReplacer(":", "-", ".", "-").Replace(key)
}
