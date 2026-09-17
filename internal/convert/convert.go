// Package convert turns a file a person added into the text form of its
// contents, as Markdown, so that a Word document, a spreadsheet, a web
// page or a PDF becomes something the page renders with structure, search
// finds by its words, and the assistant can read. Structured formats are
// read here, in Go, the way rendering wraps a Markdown parser; what Go
// cannot read well (a PDF's layout, a scanned page) is left to an external
// converter the workspace names.
package convert

import (
	"errors"
	"path"
	"strings"
)

// Result is what a file became.
type Result struct {
	// Markdown is the text form of the contents, empty for an image.
	Markdown string
	// Kind names the format, for a person and for the record.
	Kind string
	// Image says the file is a picture, which has no text of its own and
	// needs a description a person wrote.
	Image bool
}

// kinds maps an extension to the built-in reader for it.
var kinds = map[string]struct {
	name string
	read func([]byte) (string, error)
}{
	"txt": {"text", plain}, "text": {"text", plain}, "log": {"text", plain},
	"md": {"markdown", plain}, "markdown": {"markdown", plain},
	"csv": {"table", csvTable}, "tsv": {"table", tsvTable},
	"json": {"code", code("json")}, "yaml": {"code", code("yaml")}, "yml": {"code", code("yaml")},
	"html": {"web page", htmlText}, "htm": {"web page", htmlText},
	"epub": {"book", epub},
	"docx": {"document", docx},
	"xlsx": {"spreadsheet", xlsx},
	"pptx": {"slides", pptx},
	"pdf":  {"pdf", pdfText},
}

var images = map[string]bool{"png": true, "jpg": true, "jpeg": true, "gif": true, "webp": true, "svg": true, "avif": true}

// Ext is a file name's extension, lower case, without the dot.
func Ext(name string) string {
	return strings.ToLower(strings.TrimPrefix(path.Ext(name), "."))
}

// Builtin says whether a file of this name can be read here, without an
// external converter.
func Builtin(name string) bool {
	ext := Ext(name)
	_, ok := kinds[ext]
	return ok || images[ext]
}

// Kind names a file's format from its name.
func Kind(name string) string {
	ext := Ext(name)
	if images[ext] {
		return "image"
	}
	if k, ok := kinds[ext]; ok {
		return k.name
	}
	if ext == "" {
		return "file"
	}
	return ext
}

// Read converts a file with the built-in readers.
func Read(name string, data []byte) (Result, error) {
	ext := Ext(name)
	if images[ext] {
		return Result{Kind: "image", Image: true}, nil
	}
	k, ok := kinds[ext]
	if !ok {
		return Result{Kind: Kind(name)}, errors.New("no built-in reader for ." + ext + " files; name a converter for it in workspace.yaml")
	}
	md, err := k.read(data)
	if err != nil {
		return Result{Kind: k.name}, err
	}
	return Result{Markdown: strings.TrimSpace(md), Kind: k.name}, nil
}
