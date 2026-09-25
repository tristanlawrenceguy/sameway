package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/design"
)

// designPage is the living styleguide: tokens, the state language, and
// every component rendered from its own examples. It is generated from the
// same manifests agents read, so it cannot drift from the code.
func (s *Server) designPage(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	b.WriteString(`<p class="sw-prose">Everything on this page is rendered live from <code>design/</code>: the tokens, the components, and their manifest examples. Agents get the same facts from <a href="/api/describe">/api/describe</a>.</p>`)
	b.WriteString(`<nav aria-label="On this page"><ul class="sw-cluster sw-plain">`)
	for _, sec := range []string{"colour", "type", "motion", "states", "components"} {
		fmt.Fprintf(&b, `<li><a class="sw-link" href="#%s">%s</a></li>`, sec, strings.ToUpper(sec[:1])+sec[1:])
	}
	b.WriteString(`</ul></nav>`)
	s.designColour(&b)
	s.designType(&b)
	s.designMotion(&b)
	s.designStates(&b)
	s.designComponents(&b)
	s.designArrangements(&b)
	s.page(w, r, "Design system", template.HTML(b.String()), pageOptions{JSONURL: "/api/describe"})
}

func (s *Server) designColour(b *strings.Builder) {
	src, _ := design.FS.ReadFile("tokens/tokens.json")
	var doc struct {
		Color map[string]map[string]string `json:"color"`
	}
	json.Unmarshal(src, &doc)
	names := make([]string, 0, len(doc.Color))
	for n := range doc.Color {
		names = append(names, n)
	}
	sort.Strings(names)
	b.WriteString(`<h2 id="colour">Colour</h2><p class="sw-prose">Three actor tones carry provenance: <strong>human</strong> (indigo), <strong>assistant</strong> (teal), <strong>system</strong> (amber). Every text role reaches 7:1 on every surface in both themes; a Go test fails the build otherwise. Swatches below show the current theme.</p>`)
	b.WriteString(`<ul class="sw-plain sw-swatches">`)
	for _, n := range names {
		fmt.Fprintf(b, `<li class="sw-swatch"><span class="sw-swatch__chip" style="background: var(--sw-color-%s)" aria-hidden="true"></span><code>--sw-color-%s</code><span class="sw-muted sw-small">light %s · dark %s</span></li>`, n, n, doc.Color[n]["light"], doc.Color[n]["dark"])
	}
	b.WriteString(`</ul>`)
}

func (s *Server) designType(b *strings.Builder) {
	b.WriteString(`<h2 id="type">Type</h2><p class="sw-prose">A single type scale keeps hierarchy readable at any size. The heading component renders it so the page outline matches what the assistant plans.</p>`)
	// Each sample is a specimen of how a level looks, kept out of the outline:
	// real h2 to h6 in a row would skip levels. The h4 above it and the code
	// line say what it is.
	b.WriteString(`<h3 id="type-levels">Heading levels</h3>`)
	for _, level := range []int{2, 3, 4, 5, 6} {
		fmt.Fprintf(b, "<div class=\"sw-example\"><h4 class=\"sw-small\">heading \u2014 level %d</h4><p class=\"sw-small sw-muted\"><code>{\"text\": \"Type scale example\", \"level\": %d}</code></p><div class=\"sw-example__render\" aria-hidden=\"true\">%s</div></div>", level, level, s.component("heading", map[string]any{"text": "Type scale example", "level": level}))
	}
}

func (s *Server) designMotion(b *strings.Builder) {
	b.WriteString(`<h2 id="motion">Motion</h2><p class="sw-prose">A single fade-and-slide animation gives feedback without distraction. Everything stops under <code>prefers-reduced-motion</code>; the change marker becomes a static ring.</p>`)
}

func (s *Server) designStates(b *strings.Builder) {
	b.WriteString(`<h2 id="states">State language</h2><p class="sw-prose">The same facts are always available three ways: as text a person reads, as colour a sighted person scans, and as attributes a machine queries.</p>`)
	b.WriteString(`<div class="sw-table-wrap" role="region" aria-label="State attributes" tabindex="0"><table class="sw-table"><caption>Attributes every page uses</caption><thead><tr><th scope="col">Fact</th><th scope="col">Attribute</th><th scope="col">Values</th></tr></thead><tbody>`)
	rows := [][3]string{
		{"Who did it", "data-actor", "human, assistant, system"},
		{"What changed in the last turn", "data-changed", "added, updated"},
		{"Is a request in flight", "data-state on [data-region]", "idle, working"},
		{"Outcome of the last request", "data-state on [data-component=status]", "idle, working, done, error"},
		{"What a component is", "data-component", "any manifest name"},
		{"Which canvas block", "data-block-id, data-block-component", "record id, component name"},
	}
	for _, r := range rows {
		fmt.Fprintf(b, `<tr><td>%s</td><td><code>%s</code></td><td>%s</td></tr>`, r[0], r[1], r[2])
	}
	b.WriteString(`</tbody></table></div><div class="sw-cluster" style="margin-top: var(--sw-space-4)">`)
	for _, tone := range []string{"human", "assistant", "system"} {
		b.WriteString(string(s.component("badge", map[string]any{"label": "Added by " + tone, "tone": tone})))
	}
	for _, state := range []string{"idle", "working", "done", "error"} {
		b.WriteString(string(s.component("status", map[string]any{"message": "State: " + state, "state": state})))
	}
	b.WriteString(`</div>`)
}

func (s *Server) designComponents(b *strings.Builder) {
	b.WriteString(`<h2 id="components">Components</h2>`)
	for _, c := range s.app.Registry.Components() {
		var a11y struct {
			Role string                         `json:"role"`
			WCAG struct{ Target, Notes string } `json:"wcag"`
		}
		json.Unmarshal(c.Manifest.A11y, &a11y)
		fmt.Fprintf(b, `<section class="sw-panel sw-stack" id="component-%s" aria-labelledby="component-%s-h"><h3 id="component-%s-h">%s <span class="sw-muted sw-small">(%s)</span></h3><p class="sw-prose">%s</p><p class="sw-small sw-muted"><strong>Role</strong> %s · <strong>WCAG</strong> %s. %s</p>`,
			c.Manifest.Name, c.Manifest.Name, c.Manifest.Name, template.HTMLEscapeString(c.Manifest.Name), c.Source,
			template.HTMLEscapeString(c.Manifest.Description), template.HTMLEscapeString(a11y.Role), template.HTMLEscapeString(a11y.WCAG.Target), template.HTMLEscapeString(a11y.WCAG.Notes))
		if u := c.Manifest.Use; u != nil {
			fmt.Fprintf(b, `<p class="sw-prose sw-small"><strong>Use when</strong> %s`, template.HTMLEscapeString(u.When))
			if u.Not != "" {
				fmt.Fprintf(b, ` <strong>Not when</strong> %s`, template.HTMLEscapeString(u.Not))
			}
			if u.With != "" {
				fmt.Fprintf(b, ` <strong>With</strong> %s`, template.HTMLEscapeString(u.With))
			}
			b.WriteString(`</p>`)
		}
		// An example is given an id, so this page can link to it, only when
		// the component takes one; one that does not would refuse it.
		var schema struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		json.Unmarshal(c.Manifest.Props, &schema)
		_, takesID := schema.Properties["id"]
		for _, ex := range c.Manifest.Examples {
			exProps := make(map[string]any, len(ex.Props)+1)
			for k, v := range ex.Props {
				exProps[k] = v
			}
			if takesID {
				exProps["id"] = c.Manifest.Name + "-" + strings.ToLower(ex.Name)
			}
			propsJSON, _ := json.Marshal(ex.Props)
			fmt.Fprintf(b, "<div class=\"sw-example\"><h4 class=\"sw-small\">%s \u2014 %s</h4><p class=\"sw-small sw-muted\"><code>%s</code></p><div class=\"sw-example__render\">%s</div></div>",
				template.HTMLEscapeString(c.Manifest.Name), template.HTMLEscapeString(ex.Name), template.HTMLEscapeString(string(propsJSON)), s.component(c.Manifest.Name, exProps))
		}
		b.WriteString(`</section>`)
	}
}

// designArrangements shows the pages of thought the assistant can apply.
func (s *Server) designArrangements(b *strings.Builder) {
	arrangements := s.app.Registry.Arrangements()
	if len(arrangements) == 0 {
		return
	}
	b.WriteString(`<h2 id="arrangements">Arrangements</h2><p class="sw-prose">A whole page for a job, thought through once: which blocks, where each sits, how wide. The assistant applies one in a single call and fills in the words.</p>`)
	for _, a := range arrangements {
		fmt.Fprintf(b, `<section class="sw-panel sw-stack" id="arrangement-%s" aria-labelledby="arrangement-%s-h"><h3 id="arrangement-%s-h">%s <span class="sw-muted sw-small">(%s)</span></h3><p class="sw-prose">%s</p>`,
			a.Name, a.Name, a.Name, template.HTMLEscapeString(a.Name), a.Source, template.HTMLEscapeString(a.Description))
		if a.Use != nil {
			fmt.Fprintf(b, `<p class="sw-prose sw-small"><strong>Use when</strong> %s <strong>Not when</strong> %s</p>`, template.HTMLEscapeString(a.Use.When), template.HTMLEscapeString(a.Use.Not))
		}
		b.WriteString(`<ol class="sw-plain sw-stack--tight" aria-label="Blocks, top to bottom">`)
		for _, blk := range a.Blocks {
			region, span := blk.Region, blk.Span
			if region == "" {
				region = "main"
			}
			if span == 0 {
				span = 6
			}
			fmt.Fprintf(b, `<li class="sw-small">%s: <strong>%s</strong>, %s, span %d</li>`, template.HTMLEscapeString(blk.Key), template.HTMLEscapeString(blk.Component), template.HTMLEscapeString(region), span)
		}
		b.WriteString(`</ol></section>`)
	}
}
