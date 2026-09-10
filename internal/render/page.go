package render

import (
	"bytes"
	_ "embed"
	"html/template"
)

//go:embed layout.html
var layoutSrc string

var layout = template.Must(template.New("layout").Funcs(Funcs).Parse(layoutSrc))

// Page is everything the site layout needs around a page body.
type Page struct {
	Site  string
	Title string
	Lang  string
	// Controls is "auto" or "visible" and lands on the root element, where
	// the quiet layer reads it. See design/foundations/quiet.md.
	Controls string
	// Nav holds rendered link components for the main navigation: the
	// person's own content, and nothing else.
	Nav []template.HTML
	// More holds secondary destinations, shown in the footer.
	More []template.HTML
	// JSONURL is the machine-readable twin of this page, if any.
	JSONURL string
	// Body is the already-rendered main content, placed after the h1.
	Body template.HTML
	// Focus is an element id to name in the skip link, such as the newest
	// message, so keyboard users can jump straight to what changed.
	Focus      string
	FocusLabel string
}

// RenderPage wraps a body in the site layout.
func RenderPage(p Page) ([]byte, error) {
	if p.Lang == "" {
		p.Lang = "en"
	}
	var buf bytes.Buffer
	if err := layout.Execute(&buf, p); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
