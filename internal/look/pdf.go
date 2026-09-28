package look

import (
	"context"
	"encoding/base64"
	"fmt"
)

// PrintPDF prints a page to PDF in the headless browser: tagged, so a
// screen reader reads its headings, lists and tables as what they are,
// with an outline from its headings, and its colours as on screen.
func PrintPDF(ctx context.Context, url string) ([]byte, error) {
	program, err := FindBrowser()
	if err != nil {
		return nil, err
	}
	b, err := launch(ctx, program)
	if err != nil {
		return nil, err
	}
	defer b.close()
	var nav struct {
		ErrorText string `json:"errorText"`
	}
	if err := b.call(ctx, "Page.navigate", map[string]any{"url": url}, &nav); err != nil {
		return nil, err
	}
	if nav.ErrorText != "" {
		return nil, fmt.Errorf("opening %s: %s", url, nav.ErrorText)
	}
	if err := b.settle(ctx); err != nil {
		return nil, err
	}
	var out struct {
		Data string `json:"data"`
	}
	if err := b.call(ctx, "Page.printToPDF", map[string]any{
		"printBackground":         true,
		"generateTaggedPDF":       true,
		"generateDocumentOutline": true,
		"preferCSSPageSize":       true,
		"marginTop":               0.6, "marginBottom": 0.6, "marginLeft": 0.6, "marginRight": 0.6,
	}, &out); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(out.Data)
}
