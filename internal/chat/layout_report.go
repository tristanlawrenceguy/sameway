package chat

import (
	"fmt"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Layout now in its parts. It was built as one string, and what needed
// only part of it (the to-look-at list at the start of a turn) cut that
// part back out by searching the words for " To look at: ", which any
// change to the wording would have broken without a word. Each check was
// another line appended where the string was built. Now the report is its
// parts, the checks are one list, and the words are made from them.

// layoutReport is how a tab reads (its rows, panes and measures), what to
// look at on it, and the arrange_canvas call that would fix the order and
// widths, "" when there is none.
type layoutReport struct {
	reads    string
	problems []string
	fix      string
}

// layoutReportOf is the report of one tab.
func (s *Service) layoutReportOf(canvas string) layoutReport {
	var all []placed
	recs := s.canvasBlocks(canvas)
	for _, b := range recs {
		all = append(all, placedOf(b))
	}
	s.measure(all, recs)
	r := layoutReport{reads: readsNow(all) + measuredHow(all), problems: s.tabProblems(all, recs, canvas)}
	if len(r.problems) > 0 {
		r.fix = suggestArrangement(all, canvas)
	}
	return r
}

// tabProblems is every check Layout now makes, in the order it says them;
// a new check is a line here.
func (s *Service) tabProblems(all []placed, recs []*store.Record, canvas string) []string {
	var out []string
	for _, found := range [][]string{
		layoutProblems(all),       // layout_rules.go
		roomProblems(all, canvas), // layout_measured.go
		s.clutterProblems(recs),   // layout_clutter.go
		copiesProblems(recs),      // layout_copies.go
		s.leftoverProblems(recs),  // layout_leftover.go
	} {
		out = append(out, found...)
	}
	return out
}

// lookAt is the to-look-at list in words, numbered, with the call that
// would fix what it can: " (1) …. (2) …. This would fix …".
func (r layoutReport) lookAt() string {
	var b strings.Builder
	for i, p := range r.problems {
		fmt.Fprintf(&b, " (%d) %s.", i+1, p)
	}
	if r.fix != "" {
		b.WriteString(" This would fix the order and widths in one change: arrange_canvas " + r.fix + ". Move or reshape what is already there when it makes the page read better, not only what you added.")
	}
	return b.String()
}

// String is the report as Layout now says it.
func (r layoutReport) String() string {
	if len(r.problems) == 0 {
		return r.reads + " It reads in order with full rows."
	}
	return r.reads + " To look at:" + r.lookAt()
}

// readsNow is the rows of the main region with how full each is, then the
// panes: "Layout now: row 1: … (6 of 12 empty); right pane: …."
func readsNow(all []placed) string {
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
	return b.String()
}
