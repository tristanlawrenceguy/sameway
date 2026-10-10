package server

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
)

// How tags an action gave are shown on Today (design/foundations/sorting.md):
// each record in one place, so a tag on an email waiting to be sorted is
// checked beside that email, not again under Tags to check; a tag kept for
// the person from their past choices is folded away as a count but never
// silent, each one changeable; what needs nothing is folded the same way.

// keptFor is how long a tag kept from the person's choices is shown.
const keptFor = 7 * 24 * time.Hour

// tagsToCheck are the suggested tags, newest first, a few at a time.
func (s *Server) tagsToCheck() []*store.Record {
	if _, ok := s.app.Types.Get(chat.ClassificationType); !ok {
		return nil
	}
	recs, _ := s.app.Store.List(chat.ClassificationType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 100})
	var out []*store.Record
	alone := s.aloneTags()
	for _, c := range recs {
		if c.Fields["state"] == "suggested" && !alone[strings.ToLower(fmt.Sprint(c.Fields["tag"]))] && len(out) < 20 {
			out = append(out, c)
		}
	}
	return out
}

// tagsOn are the tags to check by the record they are on, as type/id.
func (s *Server) tagsOn() map[string][]*store.Record {
	out := map[string][]*store.Record{}
	for _, c := range s.tagsToCheck() {
		ref := fmt.Sprint(c.Fields["record"])
		out[ref] = append(out[ref], c)
	}
	return out
}

// otherTags are the person's tags a tag can be changed to, by name.
func (s *Server) otherTags(alone map[string]bool) []any {
	var out []any
	for _, t := range s.tagNames() {
		if !alone[strings.ToLower(t)] {
			out = append(out, t)
		}
	}
	return out
}

// tagCheck is one tag on a record: what it was given and why, then the
// presses to keep it, take it off or change it. A tag already kept for
// the person is not offered Keep again.
func (s *Server) tagCheck(c, rec *store.Record, other []any) string {
	esc := template.HTMLEscapeString
	name, tag := s.nameOf(rec), fmt.Sprint(c.Fields["tag"])
	said := "Tagged"
	if c.Fields["confirmed_by"] == "judgement" {
		said = "Kept"
	}
	var b strings.Builder
	b.WriteString(`<p>` + said + ` <strong>` + esc(tag) + `</strong>`)
	if why, _ := c.Fields["why"].(string); strings.TrimSpace(why) != "" && why != "no other tag fits" {
		b.WriteString(`: ` + esc(strings.TrimSuffix(strings.TrimSpace(why), ".")))
	}
	b.WriteString(`.</p><div class="sw-cluster">`)
	id := ui.Hidden("id", c.ID)
	context := tag + " on " + name
	if said == "Tagged" {
		b.WriteString(string(s.form(ui.Form{Action: "/tags/keep", Hidden: id, Button: &ui.Button{Label: "Keep", Context: context, Variant: ui.Secondary}})))
	}
	b.WriteString(string(s.form(ui.Form{Action: "/tags/off", Hidden: id, Button: &ui.Button{Label: "Take it off", Context: context, Variant: ui.Quiet}})))
	if len(other) > 1 {
		value := other[0]
		for _, o := range other {
			if strings.EqualFold(fmt.Sprint(o), tag) {
				value = o
			}
		}
		b.WriteString(string(s.form(ui.Form{Action: "/tags/change", Class: "sw-cluster", Hidden: id,
			Body:   s.component("select", map[string]any{"label": "Change to", "context": context, "name": "to", "as": "dropdown", "options": other, "value": value}),
			Button: &ui.Button{Label: "Change", Context: context, Variant: ui.Quiet}})))
	}
	b.WriteString(`</div>`)
	return b.String()
}

// tagRow is a record's line with its tags to check under it.
func (s *Server) tagRow(rec *store.Record, cls []*store.Record, other []any) string {
	var b strings.Builder
	b.WriteString(`<li class="sw-stack"><a class="sw-link" href="/t/` + rec.Type + `/` + rec.ID + `">` + template.HTMLEscapeString(s.nameOf(rec)) + `</a>`)
	for _, c := range cls {
		b.WriteString(s.tagCheck(c, rec, other))
	}
	return b.String() + `</li>`
}

// tagSection is Tags to check on Today: the tags given to records not
// waiting to be sorted (those are checked beside their record), then the
// tags kept for the person from their choices, folded.
func (s *Server) tagSection(sorting map[string]bool) string {
	other := s.otherTags(nil)
	var b strings.Builder
	seen := map[string]bool{}
	on := s.tagsOn()
	for _, c := range s.tagsToCheck() {
		ref := fmt.Sprint(c.Fields["record"])
		if sorting[ref] || seen[ref] {
			continue
		}
		seen[ref] = true
		if rec, err := s.taggedRecord(c); err == nil {
			b.WriteString(s.tagRow(rec, on[ref], other))
		}
	}
	kept := s.keptForYou(other)
	if b.Len() == 0 && kept == "" {
		return ""
	}
	out := `<h2>Tags to check</h2>`
	if b.Len() > 0 {
		out += `<ul class="sw-plain sw-rows">` + b.String() + `</ul>`
	}
	return out + kept
}

// keptForYou are the tags kept for the person from their past choices in
// the last week, folded as a count: the action acted for them, so they
// can see what it did and change it, without it asking for attention.
func (s *Server) keptForYou(other []any) string {
	recs, _ := s.app.Store.List(chat.ClassificationType, store.ListOptions{OrderBy: "created_at", Desc: true, Limit: 100})
	since := s.now().Add(-keptFor)
	var b strings.Builder
	n := 0
	for _, c := range recs {
		if c.Fields["state"] != "confirmed" || c.Fields["confirmed_by"] != "judgement" || c.CreatedAt.Before(since) {
			continue
		}
		if rec, err := s.taggedRecord(c); err == nil {
			n++
			b.WriteString(s.tagRow(rec, []*store.Record{c}, other))
		}
	}
	return s.folded("Kept for you from your choices", n, "tag", "tags", b.String())
}

// folded is rows under a disclosure that says how many, or nothing.
func (s *Server) folded(label string, n int, one, many, rows string) string {
	if n == 0 {
		return ""
	}
	of := many
	if n == 1 {
		of = one
	}
	body, err := s.app.Registry.RenderSlot("disclosure", map[string]any{"label": label, "count": n, "of": of}, template.HTML(`<ul class="sw-plain sw-rows">`+rows+`</ul>`))
	if err != nil {
		return ""
	}
	return string(body)
}

// foldedNothing is what was looked at and needs nothing, folded away as a
// count on Today: newsletters and receipts do not compete for attention,
// and one changed to another tag teaches the action what it missed.
func (s *Server) foldedNothing(items []*store.Record, alone map[string]bool) string {
	esc := template.HTMLEscapeString
	other := s.otherTags(alone)
	var b strings.Builder
	n, tag := 0, ""
	for _, rec := range items {
		t := s.hasAlone(rec, alone)
		c := s.latestTag(rec, t)
		if t == "" || c == nil || len(other) == 0 {
			continue
		}
		n, tag = n+1, t
		name := s.nameOf(rec)
		b.WriteString(`<li class="sw-stack"><p><a class="sw-link" href="/t/` + rec.Type + `/` + rec.ID + `">` + esc(name) + `</a>`)
		if why, _ := c.Fields["why"].(string); why != "" && why != "no other tag fits" {
			b.WriteString(`: ` + esc(why))
		}
		b.WriteString(`</p>` + string(s.form(ui.Form{Action: "/tags/change", Class: "sw-cluster", Hidden: ui.Hidden("id", c.ID),
			Body:   s.component("select", map[string]any{"label": "Change to", "context": name, "name": "to", "as": "dropdown", "options": other, "value": other[0]}),
			Button: &ui.Button{Label: "Change", Context: name, Variant: ui.Quiet}})) + `</li>`)
	}
	if n == 0 {
		return ""
	}
	return s.folded(strings.ToUpper(tag[:1])+tag[1:], n, "item", "items", b.String())
}
