package render_test

import (
	"testing"

	"golang.org/x/net/html"

	"github.com/tristanlawrenceguy/sameway/internal/render"
	"github.com/tristanlawrenceguy/sameway/internal/render/htmltest"
)

// findAlertExamples returns the alert component's examples from a registry.
func findAlertExamples(reg *render.Registry) []render.Example {
	for _, c := range reg.Components() {
		if c.Manifest.Name == "alert" {
			return c.Manifest.Examples
		}
	}
	return nil
}

// checkNoAriaLabelAndAriaHidden walks parsed DOM, reports any element with both
// aria-label and aria-hidden.
func checkNoAriaLabelAndAriaHidden(t *testing.T, where, out string) {
	t.Helper()
	doc, err := htmltest.Parse(out)
	if err != nil {
		t.Errorf("%s: parse error: %v", where, err)
		return
	}
	doc.Walk(func(n *html.Node) {
		hasLabel := false
		hasHidden := false
		for _, a := range n.Attr {
			if a.Key == "aria-label" && a.Val != "" {
				hasLabel = true
			}
			if a.Key == "aria-hidden" {
				hasHidden = true
			}
		}
		if hasLabel && hasHidden {
			t.Errorf("%s: element <%s> has both aria-label=%q and aria-hidden=%q:\n%s", where, n.Data, getAttr(n, "aria-label"), getAttr(n, "aria-hidden"), out)
		}
	})
}

// getAttr extracts an attribute value from an HTML node.
func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
