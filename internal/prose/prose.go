// Package prose renders Markdown the way the design system needs it. The
// grammar is goldmark's, because CommonMark has more edge cases than any
// hand-rolled parser gets right without limiting what a person can write;
// the layer on top is ours, and it is what no parser knows: headings that
// fit the page outline, tables that carry a caption, nothing raw, nothing
// that runs.
package prose

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Render turns Markdown into HTML whose first heading level is base, so a
// "#" inside a block on a page that already has an h1 becomes an h2, and
// inside a card whose title is an h3 becomes an h4. Raw HTML is escaped
// and links to javascript: and the like are dropped by the parser; a table
// gets its caption from a "Table: ..." line just above it, or a caption
// for assistive technology alone when there is none.
func Render(markdown string, base int) template.HTML {
	if base < 1 {
		base = 2
	}
	src := []byte(strings.ReplaceAll(markdown, "\r\n", "\n"))
	shift := &outline{base: base}
	md := goldmark.New(
		goldmark.WithExtensions(extension.Table, extension.Strikethrough),
		goldmark.WithParserOptions(parser.WithASTTransformers(util.Prioritized(shift, 100))),
	)
	var buf bytes.Buffer
	if err := md.Convert(src, &buf); err != nil {
		return template.HTML("<p>" + template.HTMLEscapeString(markdown) + "</p>")
	}
	return template.HTML(captioned(buf.String(), shift.captions))
}

// outline shifts heading levels to fit the page, and lifts "Table: ..."
// paragraphs out as captions for the tables that follow them.
type outline struct {
	base     int
	captions []string
}

func (o *outline) Transform(doc *ast.Document, reader text.Reader, _ parser.Context) {
	src := reader.Source()
	var remove []ast.Node
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok {
			if h.Level = h.Level + o.base - 1; h.Level > 6 {
				h.Level = 6
			}
		}
		if n.Kind().String() == "Table" {
			caption := ""
			if p, ok := n.PreviousSibling().(*ast.Paragraph); ok {
				if c, ok := strings.CutPrefix(lines(p, src), "Table:"); ok {
					caption = strings.TrimSpace(c)
					remove = append(remove, p)
				}
			}
			o.captions = append(o.captions, caption)
		}
		return ast.WalkContinue, nil
	})
	for _, n := range remove {
		n.Parent().RemoveChild(n.Parent(), n)
	}
}

// lines is a block's own text, as written.
func lines(n ast.Node, src []byte) string {
	var b strings.Builder
	segments := n.Lines()
	for i := 0; i < segments.Len(); i++ {
		seg := segments.At(i)
		b.Write(seg.Value(src))
	}
	return strings.TrimSpace(b.String())
}

var tableOpen = regexp.MustCompile(`<table>`)

// captioned gives every table its caption, in order.
func captioned(html string, captions []string) string {
	i := 0
	return tableOpen.ReplaceAllStringFunc(html, func(string) string {
		caption := ""
		if i < len(captions) {
			caption = captions[i]
		}
		i++
		if caption == "" {
			return `<table><caption class="sw-visually-hidden">Table</caption>`
		}
		return "<table><caption>" + template.HTMLEscapeString(caption) + "</caption>"
	})
}
