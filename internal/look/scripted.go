package look

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// helpers is what look puts in the page; see helpers.js.
//
//go:embed helpers.js
var helpers string

// Step is one thing a person does to a page before it is read: press a
// control by its name, type into the focused field or one named, press a
// key, or wait. Names match the way a screen reader says them, exactly
// first, then as part of the name.
type Step struct {
	Press string `json:"press,omitempty"`
	Type  string `json:"type,omitempty"`
	Into  string `json:"into,omitempty"`
	Key   string `json:"key,omitempty"`
	Wait  int    `json:"wait,omitempty"`
}

// Run is a page as it stood with its scripts run and the steps done.
type Run struct {
	// URL is where the page ended, which a step may have changed.
	URL string
	// HTML is the page as it stands, with what is not shown marked
	// data-look-unseen and each field's value written in, so the same
	// reader reads it.
	HTML string
	// Did says what each step reached.
	Did []string
	// Focused is what had focus after the steps.
	Focused string
	// FocusOrder is every stop Tab reaches, from the top of the page, in
	// order, pressed for real.
	FocusOrder []string
	// Errors is what went wrong on the page: exceptions, console errors,
	// failed requests.
	Errors []string
}

// Scripted opens url in a headless browser, lets its scripts run, does
// the steps, and reads the page as it then stands.
func Scripted(ctx context.Context, url string, steps []Step) (*Run, error) {
	program, err := FindBrowser()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
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
	run := &Run{}
	for i, s := range steps {
		did, err := b.do(ctx, s)
		if err != nil {
			return nil, fmt.Errorf("step %d: %w", i+1, err)
		}
		run.Did = append(run.Did, did)
		if err := b.settle(ctx); err != nil {
			return nil, err
		}
	}
	if err := b.eval(ctx, "location.pathname + location.search + location.hash", &run.URL); err != nil {
		return nil, err
	}
	if err := b.eval(ctx, "__look.focused()", &run.Focused); err != nil {
		return nil, err
	}
	if err := b.eval(ctx, "__look.snapshot()", &run.HTML); err != nil {
		return nil, err
	}
	if run.FocusOrder, err = b.focusOrder(ctx); err != nil {
		return nil, err
	}
	run.Errors = b.pageErrors()
	return run, nil
}

// eval runs an expression in the page, the helpers in place first, and
// reads its value, waiting for it when it is a promise.
func (b *browser) eval(ctx context.Context, expr string, out any) error {
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	err := b.call(ctx, "Runtime.evaluate", map[string]any{"expression": helpers + ";" + expr, "returnByValue": true, "awaitPromise": true}, &r)
	if err != nil {
		return err
	}
	if r.Exception != nil {
		return fmt.Errorf("in the page: %s %s", r.Exception.Text, r.Exception.Exception.Description)
	}
	if out != nil && len(r.Result.Value) > 0 {
		return json.Unmarshal(r.Result.Value, out)
	}
	return nil
}

// settle waits for the page to be loaded and its scripts to have had
// their turn, through a navigation a step may have started.
func (b *browser) settle(ctx context.Context) error {
	time.Sleep(150 * time.Millisecond)
	var last error
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		var state string
		// While a new page loads there is no page to ask; ask again.
		if last = b.eval(ctx, "document.readyState", &state); last == nil && state == "complete" {
			return b.eval(ctx, "__look.settled()", nil)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	if last == nil {
		last = errors.New("it did not finish loading")
	}
	return fmt.Errorf("waiting for the page: %w", last)
}

// do does one step and says what it reached.
func (b *browser) do(ctx context.Context, s Step) (string, error) {
	switch {
	case s.Press != "":
		var t struct {
			X, Y  float64
			What  string
			Error string
		}
		if err := b.eval(ctx, "__look.target("+quote(s.Press)+", false)", &t); err != nil {
			return "", err
		}
		if t.Error != "" {
			return "", errors.New(t.Error)
		}
		for _, kind := range []string{"mouseMoved", "mousePressed", "mouseReleased"} {
			if err := b.call(ctx, "Input.dispatchMouseEvent", map[string]any{"type": kind, "x": t.X, "y": t.Y, "button": "left", "clickCount": 1}, nil); err != nil {
				return "", err
			}
		}
		return "pressed " + t.What, nil
	case s.Type != "":
		what := "the focused field"
		if s.Into != "" {
			var t struct{ What, Error string }
			if err := b.eval(ctx, "__look.focus("+quote(s.Into)+")", &t); err != nil {
				return "", err
			}
			if t.Error != "" {
				return "", errors.New(t.Error)
			}
			what = t.What
		}
		return "typed into " + what, b.call(ctx, "Input.insertText", map[string]any{"text": s.Type}, nil)
	case s.Key != "":
		return "pressed the " + s.Key + " key", b.key(ctx, s.Key)
	case s.Wait > 0:
		time.Sleep(time.Duration(min(s.Wait, 10000)) * time.Millisecond)
		return fmt.Sprintf("waited %d ms", s.Wait), nil
	}
	return "", errors.New("a step says press, type, key or wait")
}

// keys are the keys a step can press, as the browser is told them.
var keys = map[string]struct {
	code, text string
	vk         int
}{
	"Tab": {"Tab", "", 9}, "Enter": {"Enter", "\r", 13}, "Escape": {"Escape", "", 27}, "Space": {"Space", " ", 32},
	"ArrowUp": {"ArrowUp", "", 38}, "ArrowDown": {"ArrowDown", "", 40}, "ArrowLeft": {"ArrowLeft", "", 37}, "ArrowRight": {"ArrowRight", "", 39},
	"Home": {"Home", "", 36}, "End": {"End", "", 35}, "Backspace": {"Backspace", "", 8}, "Delete": {"Delete", "", 46},
}

// key presses a key, with Shift+ in front of it for Shift.
func (b *browser) key(ctx context.Context, name string) error {
	mods := 0
	if rest, ok := strings.CutPrefix(name, "Shift+"); ok {
		name, mods = rest, 8
	}
	k, ok := keys[name]
	if !ok {
		known := make([]string, 0, len(keys))
		for n := range keys {
			known = append(known, n)
		}
		return fmt.Errorf("no key %q; the keys are %s, each with Shift+ before it if wanted", name, strings.Join(known, ", "))
	}
	key := name
	if name == "Space" {
		key = " "
	}
	down := map[string]any{"type": "keyDown", "key": key, "code": k.code, "windowsVirtualKeyCode": k.vk, "modifiers": mods}
	if k.text != "" {
		down["text"] = k.text
	}
	if err := b.call(ctx, "Input.dispatchKeyEvent", down, nil); err != nil {
		return err
	}
	return b.call(ctx, "Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": key, "code": k.code, "windowsVirtualKeyCode": k.vk, "modifiers": mods}, nil)
}

// focusOrder presses Tab from the top of the page until it comes round
// again, and says each stop.
func (b *browser) focusOrder(ctx context.Context) ([]string, error) {
	if err := b.eval(ctx, "__look.top()", nil); err != nil {
		return nil, err
	}
	var order []string
	seen := map[int]bool{}
	for i := 0; i < 250; i++ {
		if err := b.key(ctx, "Tab"); err != nil {
			return nil, err
		}
		var at struct {
			ID   int
			Says string
		}
		if err := b.eval(ctx, "__look.at()", &at); err != nil {
			return nil, err
		}
		if at.ID == 0 {
			if len(order) > 0 {
				break
			}
			continue
		}
		if seen[at.ID] {
			break
		}
		seen[at.ID] = true
		order = append(order, at.Says)
	}
	return order, nil
}

func quote(s string) string {
	raw, _ := json.Marshal(s)
	return string(raw)
}
