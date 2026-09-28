package convert_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
)

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		io.WriteString(f, body)
	}
	w.Close()
	return buf.Bytes()
}

// Every structured format Go can read becomes Markdown with its structure
// kept: headings, lists, captioned tables, code; an image is an image.
func TestFilesBecomeStructuredText(t *testing.T) {
	docx := zipOf(t, map[string]string{"word/document.xml": `<?xml version="1.0"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>
<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>The plan</w:t></w:r></w:p>
<w:p><w:r><w:t>Three things, </w:t></w:r><w:r><w:t>in order.</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Water the garden</w:t></w:r></w:p>
<w:p><w:pPr><w:numPr><w:ilvl w:val="1"/><w:numId w:val="1"/></w:numPr></w:pPr><w:r><w:t>Front beds first</w:t></w:r></w:p>
<w:p><w:pPr><w:pStyle w:val="ListBullet"/></w:pPr><w:r><w:t>Styled bullet</w:t></w:r></w:p>
<w:p><w:pPr><w:pStyle w:val="ListNumber"/></w:pPr><w:r><w:t>Styled number</w:t></w:r></w:p>
<w:tbl><w:tr><w:tc><w:p><w:r><w:t>Thing</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Cost</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>Seeds</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>6</w:t></w:r></w:p></w:tc></w:tr></w:tbl>
</w:body></w:document>`})
	r, err := convert.Read("plan.docx", docx)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# The plan", "Three things, in order.", "- Water the garden", "  - Front beds first", "- Styled bullet", "1. Styled number\n\n| Thing | Cost |", "| Seeds | 6 |"} {
		if !strings.Contains(r.Markdown, want) {
			t.Errorf("docx: missing %q in:\n%s", want, r.Markdown)
		}
	}
	if r.Kind != "document" {
		t.Errorf("docx kind %q", r.Kind)
	}

	book := excelize.NewFile()
	book.SetSheetName("Sheet1", "Costs")
	book.SetCellValue("Costs", "A1", "Thing")
	book.SetCellValue("Costs", "B1", "Cost")
	book.SetCellValue("Costs", "A2", "Plumber")
	book.SetCellValue("Costs", "B2", 80)
	var xbuf bytes.Buffer
	book.Write(&xbuf)
	r, err = convert.Read("costs.xlsx", xbuf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Markdown, "Table: Costs") || !strings.Contains(r.Markdown, "| Plumber | 80 |") {
		t.Errorf("xlsx: sheets become captioned tables, got:\n%s", r.Markdown)
	}

	pptx := zipOf(t, map[string]string{
		"ppt/slides/slide2.xml": `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><p:cSld><p:spTree><p:sp><p:nvSpPr><p:nvPr><p:ph type="title"/></p:nvPr></p:nvSpPr><p:txBody><a:p><a:r><a:t>Second</a:t></a:r></a:p></p:txBody></p:sp><p:graphicFrame><a:graphic><a:graphicData><a:tbl><a:tr><a:tc><a:txBody><a:p><a:r><a:t>Item</a:t></a:r></a:p></a:txBody></a:tc><a:tc><a:txBody><a:p><a:r><a:t>Cost</a:t></a:r></a:p></a:txBody></a:tc></a:tr><a:tr><a:tc><a:txBody><a:p><a:r><a:t>Liner</a:t></a:r></a:p></a:txBody></a:tc><a:tc><a:txBody><a:p><a:r><a:t>80</a:t></a:r></a:p></a:txBody></a:tc></a:tr></a:tbl></a:graphicData></a:graphic></p:graphicFrame></p:spTree></p:cSld></p:sld>`,
		"ppt/slides/slide1.xml": `<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><p:cSld><p:spTree><p:sp><p:nvSpPr><p:nvPr><p:ph type="title"/></p:nvPr></p:nvSpPr><p:txBody><a:p><a:r><a:t>Welcome</a:t></a:r></a:p></p:txBody></p:sp><p:sp><p:nvSpPr><p:nvPr><p:ph type="body"/></p:nvPr></p:nvSpPr><p:txBody><a:p><a:r><a:t>First point</a:t></a:r></a:p><a:p><a:r><a:t>Second point</a:t></a:r></a:p></p:txBody></p:sp></p:spTree></p:cSld></p:sld>`,
	})
	r, err = convert.Read("talk.pptx", pptx)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(r.Markdown, "## Welcome") || !strings.Contains(r.Markdown, "- First point") || strings.Index(r.Markdown, "## Second") < strings.Index(r.Markdown, "## Welcome") {
		t.Errorf("pptx: slides in order with titles as headings, got:\n%s", r.Markdown)
	}
	if !strings.Contains(r.Markdown, "| Item | Cost |") || !strings.Contains(r.Markdown, "| Liner | 80 |") {
		t.Errorf("pptx: a table on a slide stays a table, got:\n%s", r.Markdown)
	}

	epub := zipOf(t, map[string]string{
		"META-INF/container.xml": `<container><rootfiles><rootfile full-path="OEBPS/content.opf"/></rootfiles></container>`,
		"OEBPS/content.opf":      `<package><manifest><item id="nav" href="nav.xhtml" properties="nav"/><item id="b" href="b.xhtml"/><item id="a" href="a.xhtml"/></manifest><spine><itemref idref="nav"/><itemref idref="a"/><itemref idref="b"/></spine></package>`,
		"OEBPS/nav.xhtml":        `<html><body><nav><ol><li><a href="a.xhtml">Contents entry</a></li></ol></nav></body></html>`,
		"OEBPS/a.xhtml":          `<html><body><h1>Chapter one</h1><p>It begins.</p></body></html>`,
		"OEBPS/b.xhtml":          `<html><body><h1>Chapter two</h1><p>It goes on.</p></body></html>`,
	})
	r, err = convert.Read("book.epub", epub)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.Markdown, "# Chapter one") || strings.Index(r.Markdown, "Chapter two") < strings.Index(r.Markdown, "Chapter one") || strings.Contains(r.Markdown, "Contents entry") {
		t.Errorf("epub: chapters in spine order without the table of contents, got:\n%s", r.Markdown)
	}

	r, _ = convert.Read("list.csv", []byte("Thing,Cost\nMilk,2\nEggs,3\n"))
	if !strings.Contains(r.Markdown, "| Thing | Cost |") || !strings.Contains(r.Markdown, "| Eggs | 3 |") {
		t.Errorf("csv: a table, got:\n%s", r.Markdown)
	}
	r, _ = convert.Read("page.html", []byte(`<h2>Hello</h2><p>A <a href="/t/note">link</a>.</p><ul><li>one</li></ul><table><tr><th>Item</th><th>Cost</th></tr><tr><td>Liner</td><td>80</td></tr></table>`))
	if !strings.Contains(r.Markdown, "## Hello") || !strings.Contains(r.Markdown, "[link](/t/note)") || !strings.Contains(r.Markdown, "- one") || !strings.Contains(r.Markdown, "| Liner | 80") {
		t.Errorf("html: markdown with structure, got:\n%s", r.Markdown)
	}
	r, _ = convert.Read("photo.jpg", []byte{0xff, 0xd8})
	if !r.Image || r.Kind != "image" {
		t.Errorf("an image is an image, got %+v", r)
	}
	if _, err := convert.Read("thing.xyz", nil); err == nil || !strings.Contains(err.Error(), "converter") {
		t.Errorf("an unknown format says a converter is needed, got %v", err)
	}
	if convert.Kind("notes.PDF") != "pdf" || !convert.Builtin("x.docx") || convert.Builtin("x.xyz") {
		t.Error("Kind and Builtin read the extension")
	}
}

// A converter named in the workspace answers with Markdown, as a command
// printing it or a URL returning text or JSON with a markdown field.
func TestExternalConvertersAnswerWithMarkdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scan.pdf")
	os.WriteFile(path, []byte("%PDF-1.4 pretend"), 0o644)

	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil || r.MultipartForm == nil || len(r.MultipartForm.File["files"]) == 0 {
			http.Error(w, "no file", 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"document": {"md_content": "# Scanned\n\nThe words."}}`)
	}))
	defer remote.Close()
	convert.HTTPClient = remote.Client()
	md, err := convert.External(context.Background(), remote.URL+"/v1/convert/file", "scan.pdf", path)
	if err != nil || !strings.HasPrefix(md, "# Scanned") {
		t.Errorf("a URL converter's JSON should yield its markdown, got %q %v", md, err)
	}

	cmd := "echo \"# From a command {file}\""
	md, err = convert.External(context.Background(), cmd, "scan.pdf", path)
	if err != nil || !strings.Contains(md, "# From a command") || !strings.Contains(md, "scan.pdf") {
		t.Errorf("a command converter's output is the markdown, got %q %v", md, err)
	}
	if _, err := convert.External(context.Background(), "echo no placeholder", "x", path); err == nil {
		t.Error("a command without {file} is refused")
	}
}
