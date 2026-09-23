// Package look reads a page the way a screen reader or an agent does: its
// landmarks, headings, controls and live regions, and what is wrong with it
// structurally. It is one implementation for three readers: the tests that
// hold every component to the contract, the API and MCP surfaces an agent
// verifies a page through, and the command line.
package look

import (
	"strings"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// Outline is a page or a fragment as a machine reads it.
type Outline struct {
	Title     string     `json:"title,omitempty"`
	Landmarks []Landmark `json:"landmarks,omitempty"`
	Headings  []Heading  `json:"headings,omitempty"`
	Controls  []Control  `json:"controls,omitempty"`
	Live      []Live     `json:"live,omitempty"`
	// Components is every data-component present, in document order.
	Components []string `json:"components,omitempty"`
	// Problems is what a screen reader user would be stuck on. Empty means
	// the structure holds; it is always present so an agent can check it.
	Problems []string `json:"problems"`
}

// Landmark is a region of the page with a role.
type Landmark struct {
	Role  string `json:"role"`
	Label string `json:"label,omitempty"`
}

// Heading is one heading with its level.
type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// Control is something a person can operate: what it is, what it is
// called, and where it leads or what it submits.
type Control struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Href   string `json:"href,omitempty"`
	Action string `json:"action,omitempty"`
	Method string `json:"method,omitempty"`
	// Hidden says the control is in the document but not shown: it or an
	// ancestor carries the hidden attribute, or, read with scripts run,
	// is not drawn at all.
	Hidden bool `json:"hidden,omitempty"`
	// Disabled says it cannot be operated now.
	Disabled bool `json:"disabled,omitempty"`
	// Value is what a field holds: the text in a textbox, the chosen
	// option of a listbox. Passwords and files never say.
	Value string `json:"value,omitempty"`
	// Checked says a checkbox or radio is on.
	Checked bool `json:"checked,omitempty"`
	// Form is which form on the page the control belongs to, counting
	// from 1, so the fields and the button that sends them go together.
	Form int `json:"form,omitempty"`
}

// Live is a region assistive technology announces when it changes.
type Live struct {
	ID         string `json:"id,omitempty"`
	Politeness string `json:"politeness"`
	Text       string `json:"text"`
}

// Page reads a whole page: the rules a page must hold apply.
func Page(src string) (*Outline, error) { return read(src, true) }

// Fragment reads a rendered component on its own.
func Fragment(src string) (*Outline, error) { return read(src, false) }

func read(src string, page bool) (*Outline, error) {
	doc, err := htmltest.Parse(src)
	if err != nil {
		return nil, err
	}
	o := &Outline{Problems: []string{}}
	if ts := doc.Elements("title"); len(ts) > 0 {
		o.Title = htmltest.Text(ts[0])
	}
	seenComponent := map[string]bool{}
	forms := map[*html.Node]int{}
	var walk func(n *html.Node, underHidden bool)
	walk = func(n *html.Node, underHidden bool) {
		if n.Type == html.ElementNode {
			if _, ok := htmltest.Attr(n, "hidden"); ok {
				underHidden = true
			}
			// Marked by a reading with scripts run: not drawn from here in.
			if _, ok := htmltest.Attr(n, "data-look-unseen"); ok {
				underHidden = true
			}
			if n.Data == "form" {
				forms[n] = len(forms) + 1
			}
			if c, ok := htmltest.Attr(n, "data-component"); ok && !seenComponent[c] {
				seenComponent[c] = true
				o.Components = append(o.Components, c)
			}
			if l, ok := landmark(doc, n); ok {
				o.Landmarks = append(o.Landmarks, l)
			}
			if len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
				o.Headings = append(o.Headings, Heading{Level: int(n.Data[1] - '0'), Text: htmltest.Text(n)})
			}
			if c, ok := control(doc, n, forms); ok {
				c.Hidden = underHidden
				o.Controls = append(o.Controls, c)
			}
			if l, ok := live(n); ok {
				o.Live = append(o.Live, l)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, underHidden)
		}
	}
	walk(doc.Root, false)
	o.Problems = append(o.Problems, problems(doc, o, page)...)
	return o, nil
}

func landmark(doc *htmltest.Doc, n *html.Node) (Landmark, bool) {
	role, _ := htmltest.Attr(n, "role")
	label := doc.AccessibleName(n)
	if l, ok := htmltest.Attr(n, "aria-label"); ok {
		label = l
	} else if _, ok := htmltest.Attr(n, "aria-labelledby"); !ok {
		label = ""
	}
	switch {
	case role == "banner" || (n.Data == "header" && role == ""):
		return Landmark{Role: "banner", Label: label}, true
	case role == "navigation" || (n.Data == "nav" && role == ""):
		return Landmark{Role: "navigation", Label: label}, true
	case role == "main" || (n.Data == "main" && role == ""):
		return Landmark{Role: "main", Label: label}, true
	case role == "contentinfo" || (n.Data == "footer" && role == ""):
		return Landmark{Role: "contentinfo", Label: label}, true
	case role == "complementary" || (n.Data == "aside" && role == ""):
		return Landmark{Role: "complementary", Label: label}, true
	case role == "region" || role == "form" || role == "search":
		return Landmark{Role: role, Label: label}, true
	case (n.Data == "section" || n.Data == "form") && label != "":
		return Landmark{Role: map[string]string{"section": "region", "form": "form"}[n.Data], Label: label}, true
	}
	return Landmark{}, false
}

// control names what a person can operate, with where it goes.
func control(doc *htmltest.Doc, n *html.Node, forms map[*html.Node]int) (Control, bool) {
	c := Control{Name: doc.AccessibleName(n)}
	_, c.Disabled = htmltest.Attr(n, "disabled")
	switch n.Data {
	case "a":
		href, ok := htmltest.Attr(n, "href")
		if !ok {
			return c, false
		}
		c.Kind, c.Href = "link", href
	case "button":
		c.Kind = "button"
	case "input":
		t, _ := htmltest.Attr(n, "type")
		switch t {
		case "hidden":
			return c, false
		case "submit", "button", "reset":
			c.Kind = "button"
			if v, ok := htmltest.Attr(n, "value"); ok && c.Name == "" {
				c.Name = v
			}
		case "checkbox", "radio":
			c.Kind = t
			_, c.Checked = htmltest.Attr(n, "checked")
		default:
			c.Kind = "textbox"
			if t != "password" && t != "file" {
				c.Value, _ = htmltest.Attr(n, "value")
			}
		}
	case "select":
		c.Kind = "listbox"
		c.Value = chosen(n)
	case "textarea":
		c.Kind = "textbox"
		c.Value = textOf(n)
	case "summary":
		c.Kind = "disclosure"
	default:
		// A region a script made editable is a textbox too, and holds
		// its words.
		if ce, ok := htmltest.Attr(n, "contenteditable"); ok && (ce == "" || ce == "true" || ce == "plaintext-only") {
			c.Kind, c.Value = "textbox", htmltest.Text(n)
			break
		}
		return c, false
	}
	if c.Kind != "link" {
		for p := n.Parent; p != nil; p = p.Parent {
			if p.Type == html.ElementNode && p.Data == "form" {
				c.Form = forms[p]
				c.Action, _ = htmltest.Attr(p, "action")
				c.Method, _ = htmltest.Attr(p, "method")
				c.Method = strings.ToUpper(c.Method)
				if c.Method == "" {
					c.Method = "GET"
				}
				break
			}
		}
	}
	return c, true
}

func live(n *html.Node) (Live, bool) {
	role, _ := htmltest.Attr(n, "role")
	politeness, ok := htmltest.Attr(n, "aria-live")
	switch {
	case ok && politeness != "off":
	case role == "status" || role == "log":
		politeness = "polite"
	case role == "alert":
		politeness = "assertive"
	default:
		return Live{}, false
	}
	id, _ := htmltest.Attr(n, "id")
	return Live{ID: id, Politeness: politeness, Text: htmltest.Text(n)}, true
}

// chosen is the text of the option a listbox has chosen: the one marked
// selected, or the first, as a browser shows it.
func chosen(sel *html.Node) string {
	first := ""
	var found string
	var walk func(n *html.Node) bool
	walk = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "option" {
			if first == "" {
				first = htmltest.Text(n)
			}
			if _, ok := htmltest.Attr(n, "selected"); ok {
				found = htmltest.Text(n)
				return true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if walk(c) {
				return true
			}
		}
		return false
	}
	if walk(sel) {
		return found
	}
	return first
}

// textOf is a textarea's text as written, not collapsed as prose is.
func textOf(n *html.Node) string {
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
		}
	}
	return strings.TrimPrefix(b.String(), "\n")
}
