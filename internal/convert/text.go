package convert

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/ledongthuc/pdf"
)

func plain(data []byte) (string, error) {
	return string(bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))), nil
}

func code(lang string) func([]byte) (string, error) {
	return func(data []byte) (string, error) {
		return "```" + lang + "\n" + strings.TrimRight(string(data), "\n") + "\n```\n", nil
	}
}

func csvTable(data []byte) (string, error) { return delimited(data, ',') }
func tsvTable(data []byte) (string, error) { return delimited(data, '\t') }

// delimited renders rows as one Markdown table, the first row as headers.
func delimited(data []byte, comma rune) (string, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = comma
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		return "", err
	}
	return table(rows), nil
}

// table is rows as Markdown; the first row is the header, and short rows
// are padded so the table stays a table.
func table(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	width := 0
	for _, row := range rows {
		if len(row) > width {
			width = len(row)
		}
	}
	if width == 0 {
		return ""
	}
	cell := func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(s), "|", "\\|"), "\n", " ")
	}
	var b strings.Builder
	for i, row := range rows {
		b.WriteString("|")
		for c := 0; c < width; c++ {
			v := ""
			if c < len(row) {
				v = cell(row[c])
			}
			b.WriteString(" " + v + " |")
		}
		b.WriteString("\n")
		if i == 0 {
			b.WriteString("|" + strings.Repeat(" --- |", width) + "\n")
		}
	}
	return b.String()
}

func htmlText(data []byte) (string, error) {
	return htmltomarkdown.ConvertString(string(data))
}

// epub is a zip of web pages; the OPF spine says what order they read in.
func epub(data []byte) (string, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	files := map[string]*zip.File{}
	for _, f := range z.File {
		files[f.Name] = f
	}
	read := func(name string) []byte {
		f, ok := files[name]
		if !ok {
			return nil
		}
		rc, err := f.Open()
		if err != nil {
			return nil
		}
		defer rc.Close()
		b, _ := io.ReadAll(rc)
		return b
	}
	var container struct {
		Rootfiles []struct {
			Path string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	xml.Unmarshal(read("META-INF/container.xml"), &container)
	var order []string
	if len(container.Rootfiles) > 0 {
		opfPath := container.Rootfiles[0].Path
		var opf struct {
			Items []struct {
				ID   string `xml:"id,attr"`
				Href string `xml:"href,attr"`
			} `xml:"manifest>item"`
			Spine []struct {
				IDRef string `xml:"idref,attr"`
			} `xml:"spine>itemref"`
		}
		xml.Unmarshal(read(opfPath), &opf)
		hrefs := map[string]string{}
		for _, it := range opf.Items {
			hrefs[it.ID] = it.Href
		}
		for _, s := range opf.Spine {
			if href := hrefs[s.IDRef]; href != "" {
				order = append(order, path.Join(path.Dir(opfPath), href))
			}
		}
	}
	if len(order) == 0 {
		for _, f := range z.File {
			if ext := Ext(f.Name); ext == "xhtml" || ext == "html" || ext == "htm" {
				order = append(order, f.Name)
			}
		}
	}
	var parts []string
	for _, name := range order {
		if page := read(name); page != nil {
			if md, err := htmltomarkdown.ConvertString(string(page)); err == nil && strings.TrimSpace(md) != "" {
				parts = append(parts, strings.TrimSpace(md))
			}
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("no readable pages in the book")
	}
	return strings.Join(parts, "\n\n"), nil
}

// pdfText is the words of a born-digital PDF, in the order they were
// written, one page after another. Columns and tables are not recovered
// here; that is what an external converter is for.
func pdfText(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		if t := strings.TrimSpace(text); t != "" {
			if b.Len() > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(t)
		}
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("no text in the PDF: a scanned page needs an external converter with OCR")
	}
	return b.String(), nil
}
