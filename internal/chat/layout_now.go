package chat

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
	"github.com/tristanlawrenceguy/sameway/internal/trim"
)

// The whole page, said after every write of a block. Models asked for one
// thing add one block at the bottom and stop: in the evaluations an
// "Overdue" list sat under a less urgent one, a brief calendar was squeezed
// beside a long list, two headings said "Garden notes", and rows were left
// half empty. Each block was fine; the page was not. So every write answers
// with how the page now reads, what to look at, and an arrange_canvas call
// that would fix it, and the model decides.

// placed is one block as the grid lays it out.
type placed struct {
	rec                         *store.Record
	comp, region, frame, detail string
	size                        string
	props                       map[string]any
	span                        int
	pos                         int64
	// seen is how the person's browser drew it, by device, where it did;
	// see layout_measured.go.
	seen map[string]Reading
}

func placedOf(blk *store.Record) placed {
	l := lookOf(blk)
	p := placed{rec: blk, region: l.region, frame: l.frame, span: int(l.span)}
	p.comp, _ = blk.Fields["component"].(string)
	p.props, _ = blk.Fields["props"].(map[string]any)
	p.detail, _ = p.props["detail"].(string)
	p.pos, _ = blk.Fields["position"].(int64)
	p.size, _ = blk.Fields["size"].(string)
	return p
}

// label is the block's own name, quoted: its label, caption or title.
func (p placed) label() string {
	for _, k := range []string{"label", "caption", "title", "text", "name"} {
		if s, ok := p.props[k].(string); ok && strings.TrimSpace(s) != "" {
			return `"` + trim.Title(trim.Line(s, 32)) + `"`
		}
	}
	return ""
}

// name is what a person would call the block: its label, else what it
// is and what it shows, such as chart of entry.
func (p placed) name() string {
	if l := p.label(); l != "" {
		return l
	}
	if t := p.shows(); t != "" && t != "all" {
		return p.comp + " of " + t
	}
	return p.comp
}

// called is its name and what it is: "Overdue" collection, or chart of
// entry.
func (p placed) called() string {
	if l := p.label(); l != "" {
		return l + " " + p.comp
	}
	return p.name()
}

// shows is the kind of record a block shows, "" when it shows none.
func (p placed) shows() string {
	t, _ := p.props["type"].(string)
	return t
}

// tall says a block is usually taller than a card, short that it is a line
// or two; everything else is in between.
func (p placed) tall() bool {
	small := p.detail == "glance" || p.detail == "brief"
	switch p.comp {
	case ComponentName, "table":
		return true
	case "calendar", "tracker", "chart":
		return !small
	case "collection":
		as, _ := p.props["as"].(string)
		return as == "board"
	}
	return false
}

func (p placed) short() bool {
	switch p.comp {
	case "heading", "button", "link", "badge", "status", "alert", "search", "card", "clock":
		return true
	case "text":
		s, _ := p.props["content"].(string)
		return len(s) < 240
	}
	return false
}

// heading is the heading a block puts in the page outline: its level (2
// unless set, since a block sits under the page's h1) and its words.
func (p placed) heading() (int, string, bool) {
	key := map[string]string{"heading": "text", "collection": "label", "list": "label", "card": "title", "alert": "title"}[p.comp]
	if key == "" {
		return 0, "", false
	}
	text, _ := p.props[key].(string)
	if text == "" && p.comp == "collection" {
		text = p.shows()
	}
	if strings.TrimSpace(text) == "" {
		return 0, "", false
	}
	level := 2
	switch v := p.props["level"].(type) {
	case int64:
		level = int(v)
	case float64:
		level = int(v)
	case int:
		level = v
	}
	return level, strings.TrimSpace(text), true
}

// canvasBlocks is the blocks on one tab, in the order the page gives them.
func (s *Service) canvasBlocks(canvas string) []*store.Record {
	blocks, err := s.Store.List(BlockType, store.ListOptions{OrderBy: "position"})
	if err != nil {
		return nil
	}
	return OnCanvas(blocks, canvas)
}

// region is the blocks of one region, in order.
func region(all []placed, name string) []placed {
	var out []placed
	for _, p := range all {
		if p.region == name {
			out = append(out, p)
		}
	}
	return out
}

// gridRows falls the main region into rows of twelve, the way the grid
// does: a block that does not fit starts the next row.
func gridRows(main []placed) [][]placed {
	var rows [][]placed
	used := 0
	for _, p := range main {
		if len(rows) == 0 || used+p.span > 12 {
			rows = append(rows, nil)
			used = 0
		}
		rows[len(rows)-1] = append(rows[len(rows)-1], p)
		used += p.span
	}
	return rows
}

func fill(row []placed) int {
	n := 0
	for _, p := range row {
		n += p.span
	}
	return n
}

// LayoutNow says how a tab reads after a write: its rows with how full
// each is, its panes, what to look at, and an arrange_canvas call that
// would fix what it can. It never refuses anything; it is for the one
// building to read.
func (s *Service) LayoutNow(canvas string) string {
	var all []placed
	recs := s.canvasBlocks(canvas)
	for _, b := range recs {
		all = append(all, placedOf(b))
	}
	s.measure(all, recs)
	main := region(all, "main")
	var b strings.Builder
	b.WriteString("Layout now")
	if len(main) == 0 {
		b.WriteString(": the main region is empty")
	}
	for i, row := range gridRows(main) {
		var parts []string
		for _, p := range row {
			part := fmt.Sprintf("%s %d", p.called(), p.span)
			if r, ok := p.wide(); ok {
				part += fmt.Sprintf(" (%spx)", px(r.Height))
			}
			parts = append(parts, part)
		}
		sep := ", "
		if i == 0 {
			sep = ": "
		}
		fmt.Fprintf(&b, "%srow %d: %s", sep, i+1, strings.Join(parts, " + "))
		if n := fill(row); n < 12 {
			fmt.Fprintf(&b, " (%d of 12 empty)", 12-n)
		}
	}
	for _, pane := range []string{"left", "right"} {
		if in := region(all, pane); len(in) > 0 {
			var names []string
			for _, p := range in {
				names = append(names, p.called())
			}
			fmt.Fprintf(&b, "; %s pane: %s", pane, strings.Join(names, ", "))
		}
	}
	b.WriteString(".")
	b.WriteString(measuredHow(all))
	problems := append(layoutProblems(all), roomProblems(all, canvas)...)
	if len(problems) == 0 {
		b.WriteString(" It reads in order with full rows.")
		return b.String()
	}
	b.WriteString(" To look at: ")
	for i, p := range problems {
		fmt.Fprintf(&b, "(%d) %s. ", i+1, p)
	}
	if call := suggestArrangement(all, canvas); call != "" {
		b.WriteString("This would fix the order and widths in one change: arrange_canvas " + call + ". Move or reshape what is already there when it makes the page read better, not only what you added.")
	}
	return strings.TrimSpace(b.String())
}

// byPosition sorts blocks the way the page orders them, keeping the order
// they came in for equal positions.
func byPosition(blocks []*store.Record) {
	sort.SliceStable(blocks, func(i, j int) bool {
		a, _ := blocks[i].Fields["position"].(int64)
		b, _ := blocks[j].Fields["position"].(int64)
		return a < b
	})
}
