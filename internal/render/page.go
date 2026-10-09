package render

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/tristanlawrenceguy/sameway/internal/prose"
	"html/template"
	"strings"
	"time"
)

//go:embed layout.html
var layoutSrc string

var layout = template.Must(template.New("layout").Funcs(Funcs).Parse(layoutSrc))

// Page is everything the site layout needs around a page body.
type Page struct {
	Site  string
	Title string
	// Said is the window's title when it says more than the heading.
	Said string
	Lang string
	// Controls is "auto" or "visible" and lands on the root element, where
	// the quiet layer reads it. See design/foundations/quiet.md.
	Controls string
	// Text and Spacing are the person's reading comfort, from the
	// workspace: larger words, wider spacing (23-comfort.css).
	Text, Spacing string
	// Pace is how changes arrive, from the workspace: calm, quick or still.
	// It lands on the root element, where the motion rules read it.
	Pace string
	// Hours24 is whether the person reads times on the 24-hour clock
	// (when.TwentyFour), for the script that keeps times true.
	Hours24 bool
	// Nav holds the main navigation: the person's own content, and nothing
	// else, each list with the colour of its dot.
	Nav []NavItem
	// More holds secondary destinations, shown in the footer.
	More []template.HTML
	// JSONURL is the machine-readable twin of this page, if any.
	JSONURL string
	// Kicker sits above the title (the way here, as crumbs) and Lede under
	// it (the few facts worth knowing before reading), both rendered.
	Kicker template.HTML
	Lede   template.HTML
	// Dot is the colour of the list this page is about (1 to 6), drawn
	// before the title; 0 for none.
	Dot int
	// Developer shows the links meant for whoever builds on the workspace
	// (the guide for agents); off, the page still says where the guide is
	// in its head, so an agent finds it without a link a person must see.
	Developer bool
	// Body is the already-rendered main content, placed after the h1.
	Body template.HTML
	// EditControls is one of each design-system control, in a template the
	// page does not show, on pages where something can be edited in place:
	// the inline editor copies these rather than making its own.
	EditControls template.HTML
	// Outcome is what the person's last action came to, said once, first
	// thing under the heading on whatever page they are back on.
	Outcome template.HTML
	// Left and Right are full height panes beside the main region. Their
	// presence turns the page into an application shell, where only the
	// middle scrolls.
	Left  template.HTML
	Right template.HTML
	// Header and Footer are blocks placed in the bars at the top and the
	// bottom: what a person reaches for on every page, such as search.
	Header template.HTML
	Footer template.HTML
	// Present says who else is in the workspace just now, and where; empty
	// when nobody else is, which is most of the time.
	Present template.HTML
	// Focus is an element id to name in the skip link, such as the newest
	// message, so keyboard users can jump straight to what changed.
	Focus      string
	FocusLabel string
	// QuietTitle hides the h1 visually while keeping it in the outline.
	QuietTitle bool
	// Shell is "app" for a full height layout whose middle column scrolls,
	// and empty for an ordinary document that scrolls as a whole.
	Shell string
	// ExtraScripts are additional <script> tags rendered in the head after
	// sameway.js. Used by detail pages to load per-page scripts like 09-edit.js.
	ExtraScripts []template.HTML
}

// RenderPage wraps a body in the site layout.
func RenderPage(p Page) ([]byte, error) {
	if p.Lang == "" {
		p.Lang = "en"
	}
	if p.Left != "" || p.Right != "" {
		p.Shell = "app"
	}
	var buf bytes.Buffer
	if err := layout.Execute(&buf, p); err != nil {
		return nil, err
	}
	return []byte(prose.Outward(buf.String())), nil // a link to another site opens in a new tab
}

// NavItem is one list in the sidebar: the rendered link, and which of the
// six list colours its dot takes (0 for none).
type NavItem struct {
	HTML template.HTML
	Dot  int
	// Sub is the list's own records, listed under it (ui.nest).
	Sub []template.HTML
}

// Failed is a page that says something went wrong: its window title starts
// Error:, the first thing a screen reader says when it arrives.
func (p Page) Failed() bool {
	return strings.Contains(string(p.Outcome), `data-outcome="failed"`)
}

// TimeAttrs says on the root element how this computer tells the time, for
// the script that keeps a page's days true while it is open
// (36-days.js): its offset from UTC in minutes, its today, and the 24-hour
// clock when that is the person's.
func (p Page) TimeAttrs() template.HTMLAttr {
	now := time.Now()
	_, off := now.Zone()
	a := fmt.Sprintf(` data-zone="%d" data-today="%s"`, off/60, now.Format("2006-01-02"))
	if p.Hours24 {
		a += ` data-clock="24"`
	}
	return template.HTMLAttr(a)
}
