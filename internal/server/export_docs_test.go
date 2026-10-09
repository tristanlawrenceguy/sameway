package server_test

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

// A record goes out as a document: Markdown with its facts and words, a
// web page that stands on its own with no controls, a Word document with
// its title, language, headings, lists, links and table header as Word
// knows them, and a tagged PDF where a browser can print one.
func TestARecordGoesOutAsADocument(t *testing.T) {
	t.Parallel()
	a, h := newApp(t)
	note, err := a.Store.Create("note", map[string]any{"title": "Pond plan", "tags": []any{"garden"}, "body": "## Steps\n\n1. Dig\n2. Line it\n   - with sand\n\nSee [the guide](https://example.com/pond).\n\n| Job | Who |\n|---|---|\n| Dig | Hana |\n"})
	if err != nil {
		t.Fatal(err)
	}
	page := get(t, h, "/t/note/"+note.ID).Body.String()
	if !strings.Contains(page, `href="/export/note/`+note.ID+`.docx" download data-format="docx">Word (DOCX, `) || !strings.Contains(page, ".pdf") {
		t.Error("the record's page offers it as a document")
	}

	md := get(t, h, "/export/note/"+note.ID+".md")
	if md.Header().Get("Content-Type") != "text/markdown; charset=utf-8" || !strings.HasPrefix(md.Body.String(), "# Pond plan\n\n") || !strings.Contains(md.Body.String(), "- **Tags:** garden") || !strings.Contains(md.Body.String(), "## Steps") {
		t.Errorf("Markdown with its title, facts and words: %q", md.Body.String())
	}

	html := get(t, h, "/export/note/"+note.ID+".html").Body.String()
	for _, want := range []string{"<!doctype html>", `<html lang="en">`, "<title>Pond plan</title>", "<style>", "Line it"} {
		if !strings.Contains(html, want) {
			t.Errorf("the web page should have %s", want)
		}
	}
	if strings.Contains(html, "<form") || strings.Contains(html, "<script") || strings.Contains(html, `class="sw-bar`) {
		t.Error("the web page has no controls and no scripts")
	}

	docx := get(t, h, "/export/note/"+note.ID+".docx")
	z, err := zip.NewReader(bytes.NewReader(docx.Body.Bytes()), int64(docx.Body.Len()))
	if err != nil {
		t.Fatalf("a Word document is a package: %v", err)
	}
	read := func(name string) string {
		for _, f := range z.File {
			if f.Name == name {
				r, _ := f.Open()
				b, _ := io.ReadAll(r)
				return string(b)
			}
		}
		t.Fatalf("the package has no %s", name)
		return ""
	}
	doc := read("word/document.xml")
	for _, want := range []string{`<w:pStyle w:val="Title"/>`, `Pond plan`, `<w:pStyle w:val="Heading2"/>`, `<w:numId w:val="2"/>`, `<w:ilvl w:val="1"/><w:numId w:val="1"/>`, `<w:hyperlink r:id="link1">`, `<w:tblHeader/>`} {
		if !strings.Contains(doc, want) {
			t.Errorf("the document should have %s:\n%.3000s", want, doc)
		}
	}
	if !strings.Contains(read("docProps/core.xml"), "<dc:title>Pond plan</dc:title>") || !strings.Contains(read("word/styles.xml"), `<w:lang w:val="en"/>`) || !strings.Contains(read("word/_rels/document.xml.rels"), `Target="https://example.com/pond"`) {
		t.Error("its title, language and link are set where Word looks for them")
	}

	pdf := get(t, h, "/export/note/"+note.ID+".pdf")
	switch pdf.Code {
	case 501:
		if !strings.Contains(pdf.Body.String(), "Chrome or Edge") {
			t.Errorf("with no browser, a PDF says why and what to do: %q", pdf.Body.String())
		}
	case 200:
		b := pdf.Body.Bytes()
		if !bytes.HasPrefix(b, []byte("%PDF-")) || !bytes.Contains(b, []byte("/StructTreeRoot")) || !bytes.Contains(b, []byte("/Outlines")) {
			t.Errorf("a tagged PDF with an outline: %.200q", b)
		}
	default:
		t.Errorf("a PDF, or why not: %d %.300s", pdf.Code, pdf.Body.String())
	}
}
