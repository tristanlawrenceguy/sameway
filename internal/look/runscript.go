package look

import (
	"context"
	"encoding/json"
	"fmt"
)

// RunScript opens a page in the headless browser, waits for it to load,
// runs a script there to its end (awaiting it when it is a promise) and
// answers what it gave: work that needs a browser and no person, such as
// decoding a recording's sound on the computer that hosts the workspace.
func RunScript(ctx context.Context, url, script string) (json.RawMessage, error) {
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
	var out json.RawMessage
	if err := b.eval(ctx, script, &out); err != nil {
		return nil, err
	}
	return out, nil
}
