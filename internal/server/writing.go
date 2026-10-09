package server

import (
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/export"
	"github.com/tristanlawrenceguy/sameway/internal/prose"
	"github.com/tristanlawrenceguy/sameway/internal/records"
	"github.com/tristanlawrenceguy/sameway/internal/schema"
	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/ui"
	"github.com/tristanlawrenceguy/sameway/internal/web"
)

// Longer writing on its pages (records/pieces.go): a piece shows its outline
// (each part with what it is about, where it stands and how long it is,
// moved up or down by whoever may change it), its words against what it
// aims for, and a way to read it all as one; a part says where it is in
// the piece, with the ones either side; and both show their material,
// what is kept for that writing and never part of it.

// writingOn is what a page of writing says of its piece, parts and
// material. Each is a part of the page (parts.go), off until the assistant
// sees a reason and hands over ?show=<key>, or the person keeps it on.
func (s *Server) writingOn(r *http.Request, t *schema.Type, rec *store.Record) string {
	_, here := s.shown(r)
	page := "/t/" + t.Name + "/" + rec.ID
	var b strings.Builder
	part := func(key, what string, html string) {
		if html != "" && s.showing(r, key) {
			b.WriteString(html + s.fewer(page, key, what, here))
		}
	}
	part(ContentsPart, "contents", s.contents(t, rec))
	if !records.Organised(t) {
		return b.String()
	}
	part(PlacePart, "where it is in its piece", s.placeInPiece(t, rec))
	if parts := records.Parts(s.app.Store, t, rec); len(parts) > 0 {
		part(OutlinePart, "the outline", s.outline(r, t, rec, parts))
	}
	part(MaterialPart, "the material", s.materialOf(t, rec))
	return b.String()
}

// placeInPiece is "Part 2 of 5 of The Pond", with the parts either side.
func (s *Server) placeInPiece(t *schema.Type, rec *store.Record) string {
	piece := records.PieceOf(s.app.Store, t, rec)
	if piece == nil {
		return ""
	}
	parts := records.Parts(s.app.Store, t, piece)
	at := slices.IndexFunc(parts, func(p *store.Record) bool { return p.ID == rec.ID })
	name := template.HTMLEscapeString(s.title(t, piece))
	var b strings.Builder
	fmt.Fprintf(&b, `<nav class="sw-cluster" aria-label="Parts of %s"><p>Part %d of %d of <a class="sw-link" href="/t/%s/%s">%s</a></p>`, name, at+1, len(parts), t.Name, piece.ID, name)
	if at > 0 {
		fmt.Fprintf(&b, ` <a class="sw-link" href="/t/%s/%s" rel="prev">Before: %s</a>`, t.Name, parts[at-1].ID, template.HTMLEscapeString(s.title(t, parts[at-1])))
	}
	if at >= 0 && at < len(parts)-1 {
		fmt.Fprintf(&b, ` <a class="sw-link" href="/t/%s/%s" rel="next">Next: %s</a>`, t.Name, parts[at+1].ID, template.HTMLEscapeString(s.title(t, parts[at+1])))
	}
	b.WriteString(`</nav>`)
	return b.String()
}

// outline is a piece's parts in order, its words, and reading it all.
func (s *Server) outline(r *http.Request, t *schema.Type, piece *store.Record, parts []*store.Record) string {
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="parts"><h2 id="parts">Parts</h2><ol class="sw-stack">`)
	total := records.WordCount(s.textOf(t, piece))
	for i, p := range parts {
		words := records.WordCount(s.textOf(t, p))
		total += words
		title := s.title(t, p)
		fmt.Fprintf(&b, `<li><p><a class="sw-link" href="/t/%s/%s">%s</a></p>`, t.Name, p.ID, template.HTMLEscapeString(title))
		facts := []string{inWords(words, "word")}
		if f, ok := t.Field("status"); ok {
			if v := export.Value(*f, p.Fields["status"], s.RefTitle); v != "" {
				facts = append([]string{v}, facts...)
			}
		}
		if syn, _ := p.Fields[records.Synopsis].(string); strings.TrimSpace(syn) != "" {
			fmt.Fprintf(&b, `<p>%s</p>`, template.HTMLEscapeString(syn))
		}
		fmt.Fprintf(&b, `<p class="sw-muted">%s</p>`, strings.Join(facts, " · "))
		if web.MayChange(r) && len(parts) > 1 {
			b.WriteString(`<p class="sw-cluster">`)
			for _, mv := range []struct{ dir, label string }{{"up", "Move up"}, {"down", "Move down"}} {
				if mv.dir == "up" && i == 0 || mv.dir == "down" && i == len(parts)-1 {
					continue
				}
				b.WriteString(string(s.form(ui.Form{Action: "/t/" + t.Name + "/" + piece.ID + "/parts/move", Hidden: ui.Hidden("part", p.ID, "dir", mv.dir),
					Button: &ui.Button{Label: mv.label, Context: ": " + title, Variant: ui.Secondary}})))
			}
			b.WriteString(`</p>`)
		}
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ol>`)
	if aim, ok := piece.Fields[records.Aim].(int64); ok && aim > 0 {
		state := "going"
		if int64(total) >= aim {
			state = "met"
		}
		b.WriteString(string(s.component("meter", map[string]any{"label": "Words", "value": total, "max": aim, "state": state,
			"text": fmt.Sprintf("%d of %d words", total, aim), "words": true})))
	} else {
		fmt.Fprintf(&b, `<p>%s in all.</p>`, inWords(total, "word"))
	}
	fmt.Fprintf(&b, `<p>%s</p></section>`, s.part(ui.Link{Href: "/t/" + t.Name + "/" + piece.ID + "/whole", Label: "Read it all", Look: ui.LookButton}))
	return b.String()
}

// materialOf lists what is kept for this writing, then for what it is in.
func (s *Server) materialOf(t *schema.Type, rec *store.Record) string {
	mine, above := records.Material(s.app.Store, t, rec)
	if len(mine) == 0 && len(above) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<section class="sw-stack" aria-labelledby="material"><h2 id="material">Material</h2><p class="sw-muted">Kept with this writing and never part of it: guidelines, details, research.</p>`)
	list := func(heading string, recs []*store.Record) {
		if len(recs) == 0 {
			return
		}
		if heading != "" {
			fmt.Fprintf(&b, `<h3>%s</h3>`, template.HTMLEscapeString(heading))
		}
		b.WriteString(`<ul>`)
		for _, m := range recs {
			fmt.Fprintf(&b, `<li><a class="sw-link" href="/t/%s/%s">%s</a></li>`, t.Name, m.ID, template.HTMLEscapeString(s.title(t, m)))
		}
		b.WriteString(`</ul>`)
	}
	list("", mine)
	for p := records.PieceOf(s.app.Store, t, rec); p != nil; p = records.PieceOf(s.app.Store, t, p) {
		list("For all of "+s.title(t, p), above[p.ID])
		if len(above) == 0 {
			break
		}
		delete(above, p.ID)
	}
	b.WriteString(`</section>`)
	return b.String()
}

// textOf is a record's main writing.
func (s *Server) textOf(t *schema.Type, rec *store.Record) string {
	for _, f := range t.Fields {
		if f.Type == "markdown" {
			text, _ := rec.Fields[f.Name].(string)
			return text
		}
	}
	return ""
}

// contentsHeadings is how many headings make a piece long enough for its
// contents to be listed at the top.
const contentsHeadings = 3

// longHeadings are the headings of a record's writing, when it has
// enough to list.
func longHeadings(rec *store.Record, text string) []prose.Heading {
	if strings.Count(text, "#") < contentsHeadings {
		return nil
	}
	if h := prose.Headings(text, 3, anchorsOf(rec)); len(h) >= contentsHeadings {
		return h
	}
	return nil
}

func anchorsOf(rec *store.Record) string { return "h-" + rec.ID }

// bodyHTML is a record's main writing as it reads, its headings given ids
// when it is long enough for contents.
func bodyHTML(rec *store.Record, text string) template.HTML {
	if len(longHeadings(rec, text)) == 0 {
		return prose.Render(text, 2)
	}
	h, _ := prose.Anchored(text, 2, anchorsOf(rec))
	return h
}

// contents lists a long piece's headings, each leading to its place.
func (s *Server) contents(t *schema.Type, rec *store.Record) string {
	heads := longHeadings(rec, s.textOf(t, rec))
	if len(heads) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<nav class="sw-stack" aria-labelledby="contents"><h2 id="contents">Contents</h2><ol>`)
	for _, h := range heads {
		fmt.Fprintf(&b, `<li><a class="sw-link" href="#%s">%s</a></li>`, h.ID, template.HTMLEscapeString(h.Text))
	}
	b.WriteString(`</ol></nav>`)
	return b.String()
}
