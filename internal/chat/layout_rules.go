package chat

import (
	"encoding/json"
	"fmt"
	"strings"
)

// What makes a page read well, as rules a write is checked against. Each
// comes from what people do with a page: they scan headings first (the
// layer-cake pattern), look top-left first, see things near each other
// as one group (proximity), and a screen reader user moves by heading
// level and hears the page in source order, which on this grid is the
// order it is seen in (WCAG 1.3.1, 1.3.2, 2.4.6). design/foundations/
// layout.md has the sources.

// layoutProblems lists what to look at on a tab, each with its fix.
func layoutProblems(all []placed) []string {
	var out []string
	main := region(all, "main")
	for i, row := range gridRows(main) {
		n := fill(row)
		if n < 12 {
			last := row[len(row)-1]
			out = append(out, fmt.Sprintf("row %d leaves %d of 12 empty: widen %s to %d, or put a block of %d beside it", i+1, 12-n, last.name(), last.span+12-n, 12-n))
		}
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
	}
	out = append(out, outlineProblems(main)...)
	out = append(out, groupProblems(main)...)
	for _, p := range main {
		if (p.detail == "glance" || p.detail == "brief") && p.comp != "collection" {
			out = append(out, fmt.Sprintf("%s is %s, a pane size, in the main region: detail full here, or move it to the right pane", p.name(), p.detail))
		}
	}
	for _, pane := range []string{"left", "right"} {
		in := region(all, pane)
		for _, p := range in {
			if p.detail == "full" || p.detail == "page" || p.comp == "calendar" && p.detail == "" {
				out = append(out, fmt.Sprintf("%s is full size in the %s pane: detail brief or glance suits a pane", p.name(), pane))
			}
		}
		if len(in) > 4 {
			out = append(out, fmt.Sprintf("the %s pane holds %d blocks, more than a glance takes in: keep the few looked at most there", pane, len(in)))
		}
	}
	seen := map[int64]string{}
	for _, p := range all {
		if other, ok := seen[p.pos]; ok && p.region == "main" {
			out = append(out, fmt.Sprintf("%s and %s share position %d, so their order is not set", other, p.name(), p.pos))
		}
		seen[p.pos] = p.name()
	}
	return out
}

// headingSkip says where a block's heading skips a level down from the
// one before it, the page's h1 being the first: "" when none does.
func headingSkips(main []placed) map[string]string {
	out := map[string]string{}
	prev, prevName := 1, "the page title (level 1)"
	for _, p := range main {
		level, text, ok := p.heading()
		if !ok {
			continue
		}
		if level > prev+1 {
			out[p.rec.ID] = fmt.Sprintf("%q would be a level %d heading straight after %s, skipping level %d, and a screen reader user moving by heading would lose their place. Give it level %d, or put a level %d heading before it", text, level, prevName, prev+1, prev+1, prev+1)
		}
		prev, prevName = level, fmt.Sprintf("%q (level %d)", text, level)
	}
	return out
}

func outlineProblems(main []placed) []string {
	var out []string
	for _, p := range main {
		if why, ok := headingSkips(main)[p.rec.ID]; ok {
			out = append(out, strings.Replace(why, "would be", "is", 1))
		}
	}
	said := map[string]string{}
	for _, p := range main {
		_, text, ok := p.heading()
		if !ok {
			continue
		}
		key := strings.ToLower(text)
		if first, ok := said[key]; ok {
			out = append(out, fmt.Sprintf("two headings say %q (%s and %s): a heading block over a list that names itself says it twice; remove the heading block or rename one", text, first, p.comp))
		}
		said[key] = p.comp
		if p.comp == "heading" && (p.span != 12 || p.frame != "bare") {
			out = append(out, fmt.Sprintf("the section heading %q is span %d, frame %s: a heading wants span 12, frame bare, so its section starts on a new row", text, p.span, p.frame))
		}
	}
	return out
}

// urgent says a block lists what is late: the most urgent thing a person
// has, which belongs first.
func urgent(p placed) bool {
	label, _ := p.props["label"].(string)
	if strings.Contains(strings.ToLower(label), "overdue") {
		return true
	}
	where, _ := p.props["where"].([]any)
	for _, w := range where {
		if s, _ := w.(string); strings.Contains(s, "<today") || strings.Contains(s, "<now") {
			return true
		}
	}
	return false
}

func groupProblems(main []placed) []string {
	var out []string
	for i, p := range main {
		if urgent(p) {
			for _, q := range main[:i] {
				if q.shows() == p.shows() && !urgent(q) {
					out = append(out, fmt.Sprintf("%s is below %s: what is late is the most urgent, put it first", p.name(), q.name()))
					break
				}
			}
		}
		if p.shows() == "" {
			continue
		}
		for j := i + 1; j < len(main); j++ {
			if main[j].shows() != p.shows() {
				continue
			}
			other := ""
			for _, between := range main[i+1 : j] {
				if between.comp == "heading" {
					other = "" // another section: apart on purpose
					break
				}
				if other == "" && between.shows() != p.shows() {
					other = between.name()
				}
			}
			if other != "" {
				out = append(out, fmt.Sprintf("%s and %s both show %s, with %s between them: put related things together", p.name(), main[j].name(), p.shows(), other))
			}
			break
		}
	}
	return out
}

// section orders one section: its headings, then what is late, then the
// rest with each block beside the last one that shows the same kind.
func section(in []placed) []placed {
	lead := 0
	for lead < len(in) && in[lead].comp == "heading" {
		lead++
	}
	order := append([]placed{}, in[:lead]...)
	for _, p := range in[lead:] {
		if urgent(p) {
			order = append(order, p)
		}
	}
	for _, p := range in[lead:] {
		if urgent(p) {
			continue
		}
		at := len(order)
		for i := len(order) - 1; i >= lead && p.shows() != ""; i-- {
			if order[i].shows() == p.shows() {
				at = i + 1
				break
			}
		}
		order = append(order[:at], append([]placed{p}, order[at:]...)...)
	}
	return order
}

// arrangeItem is one block in an arrange_canvas call.
type arrangeItem struct {
	ID     string `json:"id"`
	Span   *int   `json:"span,omitempty"`
	Frame  string `json:"frame,omitempty"`
	Region string `json:"region,omitempty"`
	Size   string `json:"size,omitempty"`
}

// suggestArrangement is an arrange_canvas call that puts what is late
// first, related blocks together and every row full, or "" when it would
// change nothing. The model can send it as it is, or change it.
func suggestArrangement(all []placed, canvas string) string {
	main := region(all, "main")
	// Within each section, so nothing leaves the heading it sits under.
	var order []placed
	for start := 0; start < len(main); {
		end := start
		for end < len(main) && main[end].comp == "heading" {
			end++
		}
		for end < len(main) && main[end].comp != "heading" {
			end++
		}
		order = append(order, section(main[start:end])...)
		start = end
	}
	span := map[string]int{}
	frame := map[string]string{}
	for _, p := range order {
		span[p.rec.ID], frame[p.rec.ID] = p.span, p.frame
		if p.comp == "heading" {
			span[p.rec.ID], frame[p.rec.ID] = 12, "bare"
		}
	}
	shaped := make([]placed, len(order))
	for i, p := range order {
		p.span = span[p.rec.ID]
		shaped[i] = p
	}
	for _, row := range gridRows(shaped) {
		tall, short := false, false
		for _, p := range row {
			tall, short = tall || p.tall(), short || p.short()
		}
		if tall && short {
			for _, p := range row {
				if p.tall() {
					span[p.rec.ID] = 12
				}
			}
		}
	}
	for i := range shaped {
		shaped[i].span = span[shaped[i].rec.ID]
	}
	for _, row := range gridRows(shaped) {
		last := row[len(row)-1]
		span[last.rec.ID] += 12 - fill(row)
	}
	var items []arrangeItem
	changed := false
	for i, p := range order {
		it := arrangeItem{ID: p.rec.ID}
		if n := span[p.rec.ID]; n != p.span {
			it.Span, changed = &n, true
		}
		if f := frame[p.rec.ID]; f != p.frame {
			it.Frame, changed = f, true
		}
		if p.rec.ID != main[i].rec.ID {
			changed = true
		}
		items = append(items, it)
	}
	if !changed {
		return ""
	}
	for _, p := range all {
		if p.region == "left" || p.region == "right" {
			items = append(items, arrangeItem{ID: p.rec.ID})
		}
	}
	call := map[string]any{"blocks": items}
	if canvas != "" {
		call["canvas"] = canvas
	}
	raw, _ := json.Marshal(call)
	return string(raw)
}
