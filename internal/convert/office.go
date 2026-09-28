package convert

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Office files are zips of XML. Reading the parts that carry words and
// structure needs no library: a paragraph with a Heading style is a
// heading, a paragraph with numbering is a list item, a table is a table.

func unzip(data []byte) (map[string][]byte, error) {
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, f := range z.File {
		rc, err := f.Open()
		if err != nil {
			continue
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		out[f.Name] = b
	}
	return out, nil
}

// docx reads word/document.xml: paragraphs with their style and numbering,
// runs of text, and tables.
func docx(data []byte) (string, error) {
	parts, err := unzip(data)
	if err != nil {
		return "", err
	}
	body, ok := parts["word/document.xml"]
	if !ok {
		return "", fmt.Errorf("not a Word document: no word/document.xml")
	}
	var doc struct {
		Body struct {
			Blocks []wordBlock `xml:",any"`
		} `xml:"body"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return "", err
	}
	var b strings.Builder
	inList := false
	for _, blk := range doc.Body.Blocks {
		md := blk.markdown()
		if md == "" {
			continue
		}
		// A list ends with a blank line, so what follows is not swallowed
		// into its last item.
		item := strings.HasSuffix(md, "\n") && !strings.HasSuffix(md, "\n\n")
		if inList && !item {
			b.WriteString("\n")
		}
		inList = item
		b.WriteString(md)
	}
	return b.String(), nil
}

// wordBlock is a paragraph or a table in a document body.
type wordBlock struct {
	XMLName xml.Name
	// paragraph
	Style struct {
		Val string `xml:"val,attr"`
	} `xml:"pPr>pStyle"`
	Numbered *struct {
		Level struct {
			Val string `xml:"val,attr"`
		} `xml:"ilvl"`
	} `xml:"pPr>numPr"`
	Runs []struct {
		Text []string `xml:"t"`
		Tab  []string `xml:"tab"`
	} `xml:"r"`
	Links []struct {
		Runs []struct {
			Text []string `xml:"t"`
		} `xml:"r"`
	} `xml:"hyperlink"`
	// table
	Rows []struct {
		Cells []struct {
			Paragraphs []wordBlock `xml:"p"`
		} `xml:"tc"`
	} `xml:"tr"`
}

var headingStyle = regexp.MustCompile(`(?i)^(?:heading|title)\s*(\d)?`)

func (w wordBlock) text() string {
	var b strings.Builder
	for _, r := range w.Runs {
		b.WriteString(strings.Join(r.Text, ""))
		if len(r.Tab) > 0 {
			b.WriteString(" ")
		}
	}
	for _, l := range w.Links {
		for _, r := range l.Runs {
			b.WriteString(strings.Join(r.Text, ""))
		}
	}
	return strings.TrimSpace(b.String())
}

func (w wordBlock) markdown() string {
	switch w.XMLName.Local {
	case "tbl":
		var rows [][]string
		for _, r := range w.Rows {
			var cells []string
			for _, c := range r.Cells {
				var lines []string
				for _, p := range c.Paragraphs {
					if t := p.text(); t != "" {
						lines = append(lines, t)
					}
				}
				cells = append(cells, strings.Join(lines, " "))
			}
			rows = append(rows, cells)
		}
		return table(rows) + "\n"
	case "p":
		t := w.text()
		if t == "" {
			return ""
		}
		if m := headingStyle.FindStringSubmatch(w.Style.Val); m != nil {
			level := 1
			if m[1] != "" {
				level, _ = strconv.Atoi(m[1])
			}
			if strings.EqualFold(w.Style.Val, "Title") {
				level = 1
			}
			if level > 6 {
				level = 6
			}
			return strings.Repeat("#", level) + " " + t + "\n\n"
		}
		if w.Numbered != nil {
			indent, _ := strconv.Atoi(w.Numbered.Level.Val)
			return strings.Repeat("  ", indent) + "- " + t + "\n"
		}
		// Word's own list styles (List Bullet, List Number, List Paragraph)
		// carry the numbering on the style rather than the paragraph.
		if style := strings.ToLower(w.Style.Val); strings.HasPrefix(style, "list") {
			if strings.Contains(style, "number") {
				return "1. " + t + "\n"
			}
			return "- " + t + "\n"
		}
		return t + "\n\n"
	}
	return ""
}

// xlsx reads every sheet as a captioned table.
func xlsx(data []byte) (string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer f.Close()
	var b strings.Builder
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) == 0 {
			continue
		}
		fmt.Fprintf(&b, "Table: %s\n\n%s\n", sheet, table(rows))
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("no rows in the spreadsheet")
	}
	return b.String(), nil
}

// pptx reads each slide: its title as a heading, its text as paragraphs
// or list items.
func pptx(data []byte) (string, error) {
	parts, err := unzip(data)
	if err != nil {
		return "", err
	}
	var names []string
	for name := range parts {
		if strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml") {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool { return slideNumber(names[i]) < slideNumber(names[j]) })
	if len(names) == 0 {
		return "", fmt.Errorf("not a presentation: no slides")
	}
	var b strings.Builder
	for i, name := range names {
		var slide struct {
			Shapes []struct {
				Placeholder *struct {
					Type string `xml:"type,attr"`
				} `xml:"nvSpPr>nvPr>ph"`
				Paragraphs []struct {
					Runs []struct {
						Text string `xml:"t"`
					} `xml:"r"`
				} `xml:"txBody>p"`
			} `xml:"cSld>spTree>sp"`
			Tables []struct {
				Rows []struct {
					Cells []struct {
						Paragraphs []struct {
							Runs []struct {
								Text string `xml:"t"`
							} `xml:"r"`
						} `xml:"txBody>p"`
					} `xml:"tc"`
				} `xml:"graphic>graphicData>tbl>tr"`
			} `xml:"cSld>spTree>graphicFrame"`
		}
		if err := xml.Unmarshal(parts[name], &slide); err != nil {
			continue
		}
		titled := false
		var lines []string
		for _, sh := range slide.Shapes {
			isTitle := sh.Placeholder != nil && (sh.Placeholder.Type == "title" || sh.Placeholder.Type == "ctrTitle")
			for _, p := range sh.Paragraphs {
				var t strings.Builder
				for _, r := range p.Runs {
					t.WriteString(r.Text)
				}
				text := strings.TrimSpace(t.String())
				if text == "" {
					continue
				}
				if isTitle && !titled {
					lines = append(lines, "## "+text+"\n")
					titled = true
				} else {
					lines = append(lines, "- "+text)
				}
			}
		}
		for _, tbl := range slide.Tables {
			var rows [][]string
			for _, r := range tbl.Rows {
				var cells []string
				for _, c := range r.Cells {
					var t strings.Builder
					for _, p := range c.Paragraphs {
						for _, r := range p.Runs {
							t.WriteString(r.Text)
						}
						t.WriteString(" ")
					}
					cells = append(cells, strings.TrimSpace(t.String()))
				}
				rows = append(rows, cells)
			}
			if len(rows) > 0 {
				lines = append(lines, "\n"+table(rows))
			}
		}
		if !titled {
			lines = append([]string{fmt.Sprintf("## Slide %d\n", i+1)}, lines...)
		}
		b.WriteString(strings.Join(lines, "\n") + "\n\n")
	}
	return b.String(), nil
}

func slideNumber(name string) int {
	n, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, "ppt/slides/slide"), ".xml"))
	return n
}
