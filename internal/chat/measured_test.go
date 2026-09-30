package chat_test

import (
	"strings"
	"testing"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/chat"
)

// reading is a block as a browser of the given width drew it, of the block
// as it is now.
func reading(t *testing.T, svc *chat.Service, id string, width, height int) chat.Reading {
	t.Helper()
	blk, err := svc.Store.Get(chat.BlockType, id)
	if err != nil {
		t.Fatal(err)
	}
	panes := chat.Panes(svc.CanvasBlocks(""))
	return chat.Reading{Block: id, View: "tab", Device: chat.DeviceOf(width), Version: chat.MeasureVersion(blk, panes),
		At: time.Now(), Viewport: width, Columns: 12, Width: 400, Height: height}
}

// Readings are kept by block and kind of screen, written down, and stand
// only while the block is as it was drawn and for a while.
func TestReadingsAreKeptByDeviceAndGoStaleOnChange(t *testing.T) {
	svc := newFullService(t)
	var kept string
	svc.Measured = chat.NewMeasures("", func(s string) { kept = s })
	id := idOf(run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Soon"}, "span": 6}))
	desk, phone := reading(t, svc, id, 1440, 620), reading(t, svc, id, 390, 900)
	old := reading(t, svc, id, 800, 500)
	old.At = time.Now().Add(-8 * 24 * time.Hour)
	svc.Measured.Put([]chat.Reading{desk, phone, old})

	got := svc.MeasuredOn("", "")
	if len(got) != 2 || got[0].Device != "desktop" || got[0].Height != 620 || got[1].Device != "phone" || got[1].Height != 900 {
		t.Fatalf("a desktop and a phone reading should stand, the week-old tablet one not; got %+v", got)
	}
	if again := chat.NewMeasures(kept, nil); len(again.Fresh(id, "tab", desk.Version)) != 2 {
		t.Errorf("readings written down should come back: %s", kept)
	}
	if strings.Contains(kept, "Soon") {
		t.Errorf("what is kept is numbers and ids, never what a block says: %s", kept)
	}
	run(t, svc, "update_component", map[string]any{"id": id, "span": 12})
	if got := svc.MeasuredOn("", ""); len(got) != 0 {
		t.Errorf("a block given another width should need measuring again, got %+v", got)
	}
	for _, c := range []struct {
		w    int
		want string
	}{{320, "phone"}, {599, "phone"}, {600, "tablet"}, {1023, "tablet"}, {1024, "desktop"}} {
		if got := chat.DeviceOf(c.w); got != c.want {
			t.Errorf("%dpx is a %s, got %s", c.w, c.want, got)
		}
	}
}

// Layout now uses the heights the person's browser measured, says so, and
// names a block that scrolls inside with the call that gives it room;
// where nothing is measured it says the heights are estimated.
func TestLayoutNowUsesMeasuredHeights(t *testing.T) {
	svc := newFullService(t)
	svc.Measured = chat.NewMeasures("", nil)
	list := idOf(run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "task", "label": "Errands"}, "span": 6}))
	note := idOf(run(t, svc, "add_component", map[string]any{"component": "text", "props": map[string]any{"content": "Back at five."}, "span": 6}))
	before := svc.LayoutNow("")
	if !strings.Contains(before, "Heights estimated: nobody has opened this tab since it changed.") {
		t.Errorf("with nothing measured the line should say the heights are estimated:\n%s", before)
	}

	tall := reading(t, svc, list, 1440, 820)
	short := reading(t, svc, note, 1440, 140)
	onPhone := reading(t, svc, list, 390, 480)
	onPhone.Content, onPhone.Box = 1240, 480
	svc.Measured.Put([]chat.Reading{tall, short, onPhone})
	after := svc.LayoutNow("")
	for _, want := range []string{
		`row 1: "Errands" collection 6 (820px) + text 6 (140px)`,
		"Heights measured on your desktop (1,440px wide) and your phone (390px wide).",
		`"Errands" is 820px tall beside text at 140px on your desktop, leaving 680px blank under text: give "Errands" a row of its own (span 12)`,
		`"Errands" scrolls inside on your phone (1,240px of content in a 480px box): give it span 12: arrange_canvas {"blocks":[{"id":"` + list + `","span":12},{"id":"` + note + `"}]}`,
	} {
		if !strings.Contains(after, want) {
			t.Errorf("the measured line should say %q:\n%s", want, after)
		}
	}
	if strings.Contains(after, "estimated") {
		t.Errorf("every block measured, nothing is estimated:\n%s", after)
	}

	// Measured close in height, the row has no hole, whatever the kinds.
	svc.Measured.Put([]chat.Reading{reading(t, svc, list, 1440, 180), reading(t, svc, note, 1440, 150)})
	if got := svc.LayoutNow(""); strings.Contains(got, "blank under") || strings.Contains(got, "is tall beside") {
		t.Errorf("two blocks of a height leave no hole:\n%s", got)
	}
}

// A block in a pane taller than the pane, cut off, or running off the
// side of the screen is said with its fix.
func TestLayoutNowSaysWhatTheScreenCouldNotShow(t *testing.T) {
	svc := newFullService(t)
	svc.Measured = chat.NewMeasures("", nil)
	main := idOf(run(t, svc, "add_component", map[string]any{"component": "text", "props": map[string]any{"content": "Plans"}, "span": 12}))
	side := idOf(run(t, svc, "add_component", map[string]any{"component": "collection", "props": map[string]any{"type": "note", "label": "Notes"}, "region": "right", "size": "compact"}))
	inPane := reading(t, svc, side, 1440, 1240)
	inPane.Pane = 700
	wide := reading(t, svc, main, 390, 300)
	wide.Past, wide.Cut = 120, 60
	svc.Measured.Put([]chat.Reading{inPane, wide})
	got := svc.LayoutNow("")
	for _, want := range []string{
		`"Notes" is 1,240px tall in a 700px right pane on your desktop, so the pane scrolls to show it: detail brief, or move it out of the right pane, give it span 12 and the full size: arrange_canvas {"blocks":[{"id":"` + main + `"},{"id":"` + side + `","span":12,"region":"main","size":"full"}]}`,
		`text runs 120px past the right edge of the screen on your phone, so the page scrolls sideways (WCAG 1.4.10): it already has the whole width, so its component does not wrap and needs fixing`,
		`text has 60px of its content cut off on your phone, where nothing scrolls to it: it already has the whole width`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("should say %q:\n%s", want, got)
		}
	}
}
