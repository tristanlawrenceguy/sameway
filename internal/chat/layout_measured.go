package chat

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/tristanlawrenceguy/sameway/internal/store"
)

// Layout now with the page as it was drawn: rows with the heights the
// person's browser measured, and what scrolls inside a box too small for
// it, is cut off, or runs off the side of the screen, each with the
// arrange_canvas call that gives it room. Where nothing was measured the
// line says the heights are estimated. See measured.go.

// blankMin is how much blank space under a block beside a taller one is
// worth a word: about three lines of text.
const blankMin = 160

// measure gives each block the readings that still stand for it.
func (s *Service) measure(all []placed, recs []*store.Record) {
	if s.Measured == nil {
		return
	}
	panes := Panes(recs)
	for i := range all {
		all[i].seen = s.Measured.Fresh(all[i].rec.ID, "tab", MeasureVersion(all[i].rec, panes))
	}
}

// wide is the reading a row is judged by: a desktop's, else a tablet's,
// taken while the grid had its twelve columns.
func (p placed) wide() (Reading, bool) {
	for _, d := range []string{"desktop", "tablet"} {
		if r, ok := p.seen[d]; ok && r.Columns != 1 {
			return r, true
		}
	}
	return Reading{}, false
}

// measuredHow says where the heights come from, for the line's start.
func measuredHow(all []placed) string {
	var on []string
	missing := 0
	for _, d := range devices {
		for _, p := range all {
			if r, ok := p.seen[d]; ok {
				on = append(on, fmt.Sprintf("your %s (%spx wide)", d, px(r.Viewport)))
				break
			}
		}
	}
	for _, p := range all {
		if len(p.seen) == 0 {
			missing++
		}
	}
	switch {
	case len(all) == 0:
		return ""
	case len(on) == 0:
		return " Heights estimated: nobody has opened this tab since it changed."
	case missing > 0:
		return fmt.Sprintf(" Heights measured on %s; %d block(s) changed since, estimated.", strings.Join(on, " and "), missing)
	}
	return " Heights measured on " + strings.Join(on, " and ") + "."
}

// px writes a number of pixels the way it is read: 1,240.
func px(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0 && s[i-1] != '-'; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// tallestBlank is a row's tallest block and the one with most blank under
// it beside that, by measure, when every block in the row was measured.
func tallestBlank(row []placed) (tall, short placed, gap int, measured bool) {
	if len(row) < 2 {
		return tall, short, 0, false
	}
	for _, p := range row {
		if _, ok := p.wide(); !ok {
			return tall, short, 0, false
		}
	}
	h := func(p placed) int { r, _ := p.wide(); return r.Height }
	tall, short = row[0], row[0]
	for _, p := range row {
		if h(p) > h(tall) {
			tall = p
		}
		if h(p) < h(short) {
			short = p
		}
	}
	return tall, short, h(tall) - h(short), true
}

// besideProblems is what leaves a hole in a row: measured, a block much
// taller than the one beside it; else a kind that is usually tall beside
// one that is usually short.
func besideProblems(row []placed) []string {
	if tall, short, gap, ok := tallestBlank(row); ok {
		th, _ := tall.wide()
		sh, _ := short.wide()
		if gap < blankMin || sh.Height*10 > th.Height*6 {
			return nil
		}
		return []string{fmt.Sprintf("%s is %spx tall beside %s at %spx on your %s, leaving %spx blank under %s: give %s a row of its own (span 12), or put %s in the right pane",
			tall.name(), px(th.Height), short.name(), px(sh.Height), th.Device, px(gap), short.name(), tall.name(), short.name())}
	}
	var out []string
	for _, t := range row {
		if !t.tall() {
			continue
		}
		for _, sh := range row {
			if sh.short() {
				out = append(out, fmt.Sprintf("%s is tall beside %s, which leaves a hole under %s: give the %s a row of its own, or the right pane with detail brief", t.name(), sh.name(), sh.name(), t.comp))
				break
			}
		}
	}
	return out
}

// tooTall is the blocks of a row to give a row of their own.
func tooTall(row []placed) []string {
	if tall, short, gap, ok := tallestBlank(row); ok {
		th, _ := tall.wide()
		sh, _ := short.wide()
		if gap >= blankMin && sh.Height*10 <= th.Height*6 {
			return []string{tall.rec.ID}
		}
		return nil
	}
	tall, short := false, false
	for _, p := range row {
		tall, short = tall || p.tall(), short || p.short()
	}
	var out []string
	for _, p := range row {
		if tall && short && p.tall() {
			out = append(out, p.rec.ID)
		}
	}
	return out
}

// roomProblems is what the person's screen could not show whole: a block
// that scrolls inside, is cut off, runs off the side, or is taller than
// the pane it is in; each with the call that gives it room. Each kind is
// said once per block, for the smallest screen it happened on.
func roomProblems(all []placed, canvas string) []string {
	var out []string
	for _, p := range all {
		said := map[string]bool{}
		say := func(kind, s string) {
			if !said[kind] {
				said[kind] = true
				out = append(out, s)
			}
		}
		for i := len(devices) - 1; i >= 0; i-- {
			r, ok := p.seen[devices[i]]
			if !ok {
				continue
			}
			on := "on your " + r.Device
			if r.Scrolls() {
				say("scroll", fmt.Sprintf("%s scrolls inside %s (%spx of content in a %spx box): %s", p.name(), on, px(r.Content), px(r.Box), grow(all, canvas, p)))
			}
			if r.Pane > 0 && r.Height > r.Pane+1 {
				say("pane", fmt.Sprintf("%s is %spx tall in a %spx %s pane %s, so the pane scrolls to show it: detail brief, or %s", p.name(), px(r.Height), px(r.Pane), p.region, on, grow(all, canvas, p)))
			}
			if r.Cut > 0 {
				say("cut", fmt.Sprintf("%s has %spx of its content cut off %s, where nothing scrolls to it: %s", p.name(), px(r.Cut), on, grow(all, canvas, p)))
			}
			if r.Sideways > 0 {
				say("sideways", fmt.Sprintf("%s scrolls sideways inside %s (%spx hidden): %s", p.name(), on, px(r.Sideways), grow(all, canvas, p)))
			}
			if r.Past > 0 {
				fix := grow(all, canvas, p) + "; if it still does, its component does not wrap and needs fixing"
				if strings.HasPrefix(fix, whole) {
					fix = "it already has the whole width, so its component does not wrap and needs fixing"
				}
				say("past", fmt.Sprintf("%s runs %spx past the right edge of the screen %s, so the page scrolls sideways (WCAG 1.4.10): %s", p.name(), px(r.Past), on, fix))
			}
		}
	}
	return out
}

const whole = "it already has the whole width"

// grow says how a block gets more room, and the arrange_canvas call for
// it: the whole width, the full size, and the main region.
func grow(all []placed, canvas string, p placed) string {
	it := arrangeItem{ID: p.rec.ID}
	var how []string
	if p.region == "left" || p.region == "right" {
		it.Region = "main"
		how = append(how, "move it out of the "+p.region+" pane")
	}
	if p.span < 12 || it.Region == "main" {
		twelve := 12
		it.Span = &twelve
		how = append(how, "give it span 12")
	}
	if p.size == "compact" || p.size == "icon" {
		it.Size = "full"
		how = append(how, "the full size")
	}
	if len(how) == 0 {
		return whole + ": a shorter detail, or fewer records shown (limit), fits it"
	}
	var items []arrangeItem
	for _, q := range region(all, "main") {
		if q.rec.ID != p.rec.ID {
			items = append(items, arrangeItem{ID: q.rec.ID})
		} else {
			items = append(items, it)
		}
	}
	if it.Region == "main" {
		items = append(items, it)
	}
	for _, q := range all {
		if (q.region == "left" || q.region == "right") && q.rec.ID != p.rec.ID {
			items = append(items, arrangeItem{ID: q.rec.ID})
		}
	}
	call := map[string]any{"blocks": items}
	if canvas != "" {
		call["canvas"] = canvas
	}
	raw, _ := json.Marshal(call)
	return joinAnd(how) + ": arrange_canvas " + string(raw)
}

// joinAnd lists words the way they are said: a, b and c.
func joinAnd(words []string) string {
	if len(words) < 2 {
		return strings.Join(words, "")
	}
	return strings.Join(words[:len(words)-1], ", ") + " and " + words[len(words)-1]
}
