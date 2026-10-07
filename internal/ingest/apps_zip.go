package ingest

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"path"
	"regexp"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/convert"
)

// evernote reads an .enex: each note's title, its words as Markdown, its
// tags.
func evernote(data []byte) (*Brought, error) {
	var ex struct {
		Notes []struct {
			Title   string   `xml:"title"`
			Content string   `xml:"content"`
			Tags    []string `xml:"tag"`
		} `xml:"note"`
	}
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	if err := dec.Decode(&ex); err != nil || len(ex.Notes) == 0 {
		return nil, errors.New("this .enex has no notes Sameway can read")
	}
	b := newBrought("Evernote")
	for _, n := range ex.Notes {
		body, err := convert.HTMLToMarkdown(n.Content)
		if err != nil {
			body = n.Content
		}
		title := n.Title
		if strings.TrimSpace(title) == "" {
			title = firstLine(body)
		}
		b.add("note", map[string]string{"title": title, "body": strings.TrimSpace(body), "tags": strings.Join(n.Tags, ", ")})
	}
	return b, nil
}

// notionID is the id Notion adds to the name of each page it exports.
var notionID = regexp.MustCompile(`\s+[0-9a-f]{32}$`)

// zipped reads a Google Takeout zip (Tasks, Keep) or a zip of Markdown.
func zipped(data []byte) (*Brought, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.New("this zip cannot be opened: " + err.Error())
	}
	keep, md := newBrought("Google Keep"), newBrought("Markdown notes")
	var tasks *Brought
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || f.UncompressedSize64 > 32<<20 {
			continue
		}
		name := path.Base(f.Name)
		ext := strings.ToLower(path.Ext(name))
		if ext != ".json" && ext != ".md" && ext != ".markdown" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		var buf bytes.Buffer
		buf.ReadFrom(rc)
		rc.Close()
		switch {
		case ext == ".json" && strings.EqualFold(name, "Tasks.json"):
			tasks, _ = googleJSON(buf.Bytes())
		case ext == ".json" && strings.Contains(f.Name, "Keep/"):
			keepNote(keep, buf.Bytes())
		case ext != ".json":
			title := notionID.ReplaceAllString(strings.TrimSuffix(name, path.Ext(name)), "")
			body := strings.TrimSpace(buf.String())
			if strings.HasPrefix(body, "# ") {
				line, rest, _ := strings.Cut(body, "\n")
				title, body = strings.TrimSpace(strings.TrimPrefix(line, "# ")), strings.TrimSpace(rest)
			}
			md.add("note", map[string]string{"title": title, "body": body})
		}
	}
	for _, b := range []*Brought{tasks, keep, md} {
		if b != nil && len(b.Kinds) > 0 {
			return b, nil
		}
	}
	return nil, errors.New("this zip has no tasks or notes Sameway can read: it reads the Tasks and Keep folders of Google Takeout, and Markdown files")
}
