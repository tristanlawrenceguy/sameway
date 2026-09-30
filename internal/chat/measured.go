package chat

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Measured, not guessed. How tall a block is was guessed from what it is
// and its detail: a calendar is tall, a clock is short. The page is drawn
// in the person's own browser, at their window's width, their zoom, their
// text size and spacing (ui.text, ui.spacing), their fonts and their
// phone, and only there is it known how tall a block really is, and
// whether it scrolls inside a box too small for it. So the page measures
// its blocks where it is drawn (design/base/25-measure.js) and sends the
// numbers here: sizes and block ids, never what the blocks say.
// design/foundations/layout.md has the rules.

// measureStale is how long a reading stands without being taken again: a
// list grows and a calendar fills, and the numbers with them.
const measureStale = 7 * 24 * time.Hour

// measureKept caps how many readings are kept, so a workspace with many
// blocks and devices never keeps more than a small note.
const measureKept = 600

// DeviceOf is the kind of screen a width is: a phone below 600 pixels, a
// tablet below 1024, a desktop from there.
func DeviceOf(width int) string {
	switch {
	case width < 600:
		return "phone"
	case width < 1024:
		return "tablet"
	}
	return "desktop"
}

// Reading is one block as a browser drew it: numbers, and the ids that say
// which block and which page.
type Reading struct {
	Block   string    `json:"block"`
	View    string    `json:"view"` // tab: on its tab; focus: on its own page
	Device  string    `json:"device"`
	Version string    `json:"version"` // the block as it was drawn; see MeasureVersion
	At      time.Time `json:"at"`
	// The screen: its width, its pixel ratio (zoom moves it), the root
	// font size in pixels (the browser's text size and ui.text), how many
	// columns the grid had, and the workspace's text and spacing settings.
	Viewport int     `json:"viewport"`
	Ratio    float64 `json:"ratio,omitempty"`
	Font     float64 `json:"font,omitempty"`
	Columns  int     `json:"columns,omitempty"`
	Text     string  `json:"text,omitempty"`
	Spacing  string  `json:"spacing,omitempty"`
	// The block: its size and where its row starts.
	Width  int `json:"width"`
	Height int `json:"height"`
	Top    int `json:"top"`
	// Content is how tall what scrolls inside the block, not by design,
	// is, in a box Box tall; Sideways is how much of it is hidden across.
	Content  int `json:"content,omitempty"`
	Box      int `json:"box,omitempty"`
	Sideways int `json:"sideways,omitempty"`
	// Cut is how much is hidden by a box that clips and cannot scroll.
	Cut int `json:"cut,omitempty"`
	// Designed is how much is hidden in what scrolls by design, such as a
	// conversation's history or a wide table's own region.
	Designed int `json:"by_design,omitempty"`
	// Past is how far the block runs past the right edge of the screen.
	Past int `json:"past_edge,omitempty"`
	// Pane is the height of the scrolling pane the block sits in, when it
	// sits in one.
	Pane int `json:"pane,omitempty"`
}

// Scrolls says the block scrolls inside when it was not meant to.
func (r Reading) Scrolls() bool { return r.Content > r.Box+1 && r.Box > 0 }

// Measures is the latest reading of each block on each kind of screen.
type Measures struct {
	mu   sync.Mutex
	by   map[string]Reading
	keep func(string) // writes them down; nil keeps them in memory only
	now  func() time.Time
}

// NewMeasures holds readings, starting from those written down before,
// and writes them down with keep after each change.
func NewMeasures(saved string, keep func(string)) *Measures {
	m := &Measures{by: map[string]Reading{}, keep: keep, now: time.Now}
	var list []Reading
	if json.Unmarshal([]byte(saved), &list) == nil {
		for _, r := range list {
			m.by[measureKey(r)] = r
		}
	}
	return m
}

// MeasuresIn keeps readings in the store's own notes, beside the data and
// never synced to other copies: what one person's screen drew is theirs.
func MeasuresIn(st *store.Store) *Measures {
	return NewMeasures(st.Meta("measured"), func(s string) { st.SetMeta("measured", s) })
}

func measureKey(r Reading) string { return r.Block + "|" + r.View + "|" + r.Device }

// Put keeps readings, each over the last of its block, page and screen.
func (m *Measures) Put(rs []Reading) {
	if m == nil || len(rs) == 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range rs {
		m.by[measureKey(r)] = r
	}
	list := make([]Reading, 0, len(m.by))
	for k, r := range m.by {
		if m.now().Sub(r.At) > measureStale {
			delete(m.by, k)
			continue
		}
		list = append(list, r)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].At.After(list[j].At) })
	for _, r := range list[min(len(list), measureKept):] {
		delete(m.by, measureKey(r))
	}
	list = list[:min(len(list), measureKept)]
	if m.keep != nil {
		raw, _ := json.Marshal(list)
		m.keep(string(raw))
	}
}

// Fresh is a block's readings on one page that still stand, by device:
// taken of the block as it is now, and not too long ago.
func (m *Measures) Fresh(block, view, version string) map[string]Reading {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]Reading{}
	for _, d := range devices {
		r, ok := m.by[block+"|"+view+"|"+d]
		if ok && r.Version == version && m.now().Sub(r.At) <= measureStale {
			out[d] = r
		}
	}
	return out
}

// MeasureVersion names a block as it would be drawn: what it is, its
// props, and its width and place, and on a tab which panes that tab has,
// since they take width from the middle. Its position is left out: moving
// a block does not change its size. A reading of any other version is of
// a block that has since changed, and no longer stands.
func MeasureVersion(blk *store.Record, panes string) string {
	f := blk.Fields
	raw, _ := json.Marshal([]any{f["component"], f["props"], f["span"], f["region"], f["frame"], f["size"], panes})
	h := fnv.New64a()
	h.Write(raw)
	return fmt.Sprintf("%016x", h.Sum64())
}

// Panes is which side panes a tab's blocks fill: "", "left", "right" or
// "left right".
func Panes(blocks []*store.Record) string {
	has := map[string]bool{}
	for _, b := range blocks {
		has[lookOf(b).region] = true
	}
	switch {
	case has["left"] && has["right"]:
		return "left right"
	case has["left"]:
		return "left"
	case has["right"]:
		return "right"
	}
	return ""
}

// CanvasBlocks is the blocks on one tab, in page order.
func (s *Service) CanvasBlocks(canvas string) []*store.Record { return s.canvasBlocks(canvas) }

// MeasuredOn is every reading that stands for the blocks on a tab, or on
// one block's own page when focus is its id.
func (s *Service) MeasuredOn(canvas, focus string) []Reading {
	var out []Reading
	if focus != "" {
		if blk, err := s.Store.Get(BlockType, focus); err == nil {
			out = inOrder(s.Measured.Fresh(blk.ID, "focus", MeasureVersion(blk, "")))
		}
		return out
	}
	blocks := s.canvasBlocks(canvas)
	panes := Panes(blocks)
	for _, b := range blocks {
		out = append(out, inOrder(s.Measured.Fresh(b.ID, "tab", MeasureVersion(b, panes)))...)
	}
	return out
}

// devices are the kinds of screen, widest first: a row is a row only where
// the grid has its twelve columns.
var devices = []string{"desktop", "tablet", "phone"}

func inOrder(by map[string]Reading) []Reading {
	var out []Reading
	for _, d := range devices {
		if r, ok := by[d]; ok {
			out = append(out, r)
		}
	}
	return out
}
