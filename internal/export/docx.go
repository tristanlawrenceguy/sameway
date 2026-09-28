package export

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// A Word document (.docx) written from Markdown, with what makes one
// accessible in Word's own checker: its title set, its language set,
// headings as Word's heading styles (so they are its outline), lists as
// real lists, links as links, and tables with their header row marked to
// repeat on every page.

// DOCX writes Markdown as a Word document titled title, in language lang.
func DOCX(w io.Writer, title, lang, markdown string) error {
	src := []byte(markdown)
	doc := goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough)).Parser().Parse(text.NewReader(src))
	d := &docx{src: src}
	d.para("Title", d.runs(title, run{}))
	d.blocks(doc, 0)
	return d.write(w, title, lang)
}

type docx struct {
	src   []byte
	body  strings.Builder
	links []string
}

type run struct{ bold, italic, code, strike bool }

func xesc(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

// runs is text as Word runs, in one look.
func (d *docx) runs(s string, r run) string {
	if s == "" {
		return ""
	}
	var props strings.Builder
	if r.bold {
		props.WriteString("<w:b/>")
	}
	if r.italic {
		props.WriteString("<w:i/>")
	}
	if r.strike {
		props.WriteString("<w:strike/>")
	}
	if r.code {
		props.WriteString(`<w:rStyle w:val="CodeChar"/>`)
	}
	rpr := ""
	if props.Len() > 0 {
		rpr = "<w:rPr>" + props.String() + "</w:rPr>"
	}
	return `<w:r>` + rpr + `<w:t xml:space="preserve">` + xesc(s) + `</w:t></w:r>`
}

func (d *docx) para(style, content string, extra ...string) {
	d.body.WriteString(`<w:p><w:pPr><w:pStyle w:val="` + style + `"/>` + strings.Join(extra, "") + `</w:pPr>` + content + `</w:p>`)
}

// inline is a block's inline content as runs.
func (d *docx) inline(n ast.Node, r run) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Text:
			b.WriteString(d.runs(string(c.Segment.Value(d.src)), r))
			if c.HardLineBreak() {
				b.WriteString(`<w:r><w:br/></w:r>`)
			} else if c.SoftLineBreak() {
				b.WriteString(d.runs(" ", r))
			}
		case *ast.String:
			b.WriteString(d.runs(string(c.Value), r))
		case *ast.CodeSpan:
			code := r
			code.code = true
			b.WriteString(d.runs(string(c.Text(d.src)), code))
		case *ast.Emphasis:
			e := r
			if c.Level >= 2 {
				e.bold = true
			} else {
				e.italic = true
			}
			b.WriteString(d.inline(c, e))
		case *east.Strikethrough:
			s := r
			s.strike = true
			b.WriteString(d.inline(c, s))
		case *ast.Link:
			d.links = append(d.links, string(c.Destination))
			fmt.Fprintf(&b, `<w:hyperlink r:id="link%d">`, len(d.links))
			b.WriteString(strings.ReplaceAll(d.inline(c, r), "<w:r>", `<w:r><w:rPr><w:rStyle w:val="Hyperlink"/></w:rPr>`))
			b.WriteString(`</w:hyperlink>`)
		case *ast.AutoLink:
			url := string(c.URL(d.src))
			d.links = append(d.links, url)
			fmt.Fprintf(&b, `<w:hyperlink r:id="link%d">%s</w:hyperlink>`, len(d.links), strings.ReplaceAll(d.runs(url, r), "<w:r>", `<w:r><w:rPr><w:rStyle w:val="Hyperlink"/></w:rPr>`))
		case *ast.Image:
			b.WriteString(d.runs("[Picture: "+string(c.Text(d.src))+"]", r))
		default:
			b.WriteString(d.inline(c, r))
		}
	}
	return b.String()
}

// blocks writes the blocks under n; depth is how deep in lists they are.
func (d *docx) blocks(n ast.Node, depth int) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch c := c.(type) {
		case *ast.Heading:
			d.para(fmt.Sprintf("Heading%d", min(c.Level, 6)), d.inline(c, run{}))
		case *ast.Paragraph, *ast.TextBlock:
			d.para("Normal", d.inline(c, run{}))
		case *ast.List:
			num := "1"
			if c.IsOrdered() {
				num = "2"
			}
			for item := c.FirstChild(); item != nil; item = item.NextSibling() {
				first := true
				for b := item.FirstChild(); b != nil; b = b.NextSibling() {
					if l, ok := b.(*ast.List); ok {
						wrap := ast.NewDocument()
						wrap.AppendChild(wrap, cloneList(l))
						d.blocks(wrap, depth+1)
						continue
					}
					numPr := ""
					if first {
						numPr = fmt.Sprintf(`<w:numPr><w:ilvl w:val="%d"/><w:numId w:val="%s"/></w:numPr>`, min(depth, 8), num)
					} else {
						numPr = fmt.Sprintf(`<w:ind w:left="%d"/>`, 720*(depth+1))
					}
					d.para("ListParagraph", d.inline(b, run{}), numPr)
					first = false
				}
			}
		case *ast.FencedCodeBlock, *ast.CodeBlock:
			lines := c.Lines()
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				d.para("Code", d.runs(strings.TrimRight(string(seg.Value(d.src)), "\n"), run{}))
			}
		case *ast.Blockquote:
			for b := c.FirstChild(); b != nil; b = b.NextSibling() {
				d.para("Quote", d.inline(b, run{}))
			}
		case *ast.ThematicBreak:
			d.para("Normal", "", `<w:pBdr><w:bottom w:val="single" w:sz="6" w:space="1" w:color="auto"/></w:pBdr>`)
		case *east.Table:
			d.table(c)
		default:
			d.blocks(c, depth)
		}
	}
}

// cloneList moves a nested list out to be written on its own; its parent
// is done with it.
func cloneList(l *ast.List) ast.Node {
	l.Parent().RemoveChild(l.Parent(), l)
	return l
}

func (d *docx) table(t *east.Table) {
	d.body.WriteString(`<w:tbl><w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="0" w:type="auto"/><w:tblLook w:firstRow="1"/></w:tblPr>`)
	for row := t.FirstChild(); row != nil; row = row.NextSibling() {
		_, header := row.(*east.TableHeader)
		d.body.WriteString("<w:tr>")
		if header {
			d.body.WriteString(`<w:trPr><w:tblHeader/></w:trPr>`)
		}
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			d.body.WriteString(`<w:tc><w:p>` + d.inline(cell, run{bold: header}) + `</w:p></w:tc>`)
		}
		d.body.WriteString("</w:tr>")
	}
	d.body.WriteString(`</w:tbl><w:p/>`)
}
